package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
	"naimuBack/internal/courier/pricing"
	"naimuBack/internal/courier/repo"
)

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	case http.MethodGet:
		s.handleListOrders(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}

	var req struct {
		DistanceM     int               `json:"distance_m"`
		EtaSeconds    int               `json:"eta_s"`
		ClientPrice   int               `json:"client_price"`
		PaymentMethod string            `json:"payment_method"`
		Comment       *string           `json:"comment"`
		RoutePoints   []orderPointInput `json:"route_points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if len(req.RoutePoints) < 2 {
		writeError(w, http.StatusBadRequest, "at least two route points required")
		return
	}
	if req.ClientPrice < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("client price must be >= %d", s.cfg.MinPrice))
		return
	}
	if req.PaymentMethod != "cash" && req.PaymentMethod != "online" {
		writeError(w, http.StatusBadRequest, "invalid payment method")
		return
	}

	points := make([]repo.OrderPoint, 0, len(req.RoutePoints))
	for idx, p := range req.RoutePoints {
		if strings.TrimSpace(p.Address) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("route point %d missing address", idx))
			return
		}
		if p.Lat == 0 && p.Lon == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("route point %d missing coordinates", idx))
			return
		}
		points = append(points, repo.OrderPoint{
			Seq:      idx,
			Address:  strings.TrimSpace(p.Address),
			Lat:      p.Lat,
			Lon:      p.Lon,
			Entrance: nullableString(p.Entrance),
			Apt:      nullableString(p.Apt),
			Floor:    nullableString(p.Floor),
			Intercom: nullableString(p.Intercom),
			Phone:    nullableString(p.Phone),
			Comment:  nullableString(p.Comment),
		})
	}

	recommended := pricing.Recommended(req.DistanceM, s.cfg.PricePerKM, s.cfg.MinPrice)
	order := repo.Order{
		SenderID:         senderID,
		DistanceM:        req.DistanceM,
		EtaSeconds:       req.EtaSeconds,
		RecommendedPrice: recommended,
		ClientPrice:      req.ClientPrice,
		PaymentMethod:    req.PaymentMethod,
		Points:           points,
	}
	if req.Comment != nil && strings.TrimSpace(*req.Comment) != "" {
		order.Comment = sql.NullString{String: strings.TrimSpace(*req.Comment), Valid: true}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orderID, err := s.orders.Create(ctx, order)
	if err != nil {
		s.logger.Errorf("courier: create order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	resp := map[string]interface{}{
		"order_id":          orderID,
		"recommended_price": recommended,
		"status":            repo.StatusNew,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	senderID, senderErr := parseAuthID(r, "X-Sender-ID")
	courierID, courierErr := parseAuthID(r, "X-Courier-ID")
	if senderErr != nil && courierErr != nil {
		writeError(w, http.StatusUnauthorized, "missing sender or courier id")
		return
	}

	limit, offset, err := parsePaging(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var orders []repo.Order
	if senderErr == nil {
		orders, err = s.orders.ListBySender(ctx, senderID, limit, offset)
	} else {
		orders, err = s.orders.ListByCourier(ctx, courierID, limit, offset)
	}
	if err != nil {
		s.logger.Errorf("courier: list orders failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, makeOrderResponse(o))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp})
}

func (s *Server) handleOrderSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/courier/orders/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleGetOrder(w, r, id)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch parts[1] {
	case "cancel":
		s.handleCancelOrder(w, r, id)
	case "arrive":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusCourierArrived)
	case "start":
		s.handleStartOrder(w, r, id)
	case "finish":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusDelivered)
	case "confirm-cash":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusClosed)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request, orderID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: get order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"order": makeOrderResponse(order)})
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request, orderID int64) {
	actorStatus := ""
	if _, err := parseAuthID(r, "X-Sender-ID"); err == nil {
		actorStatus = lifecycle.StatusCanceledBySender
	} else if _, err := parseAuthID(r, "X-Courier-ID"); err == nil {
		actorStatus = lifecycle.StatusCanceledByCourier
	} else {
		writeError(w, http.StatusUnauthorized, "missing sender or courier id")
		return
	}

	var req struct {
		Reason *string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.orders.UpdateStatusWithNote(ctx, orderID, actorStatus, req.Reason); errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	} else if err != nil {
		s.logger.Errorf("courier: cancel order failed: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": actorStatus})
}

func (s *Server) handleStartOrder(w http.ResponseWriter, r *http.Request, orderID int64) {
	if _, err := parseAuthID(r, "X-Courier-ID"); err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stages := []string{lifecycle.StatusPickupStarted, lifecycle.StatusPickupDone, lifecycle.StatusDeliveryStarted}
	for _, status := range stages {
		if err := s.orders.UpdateStatus(ctx, orderID, status, sql.NullString{}); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "order not found")
				return
			}
			s.logger.Errorf("courier: update order status %s failed: %v", status, err)
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": lifecycle.StatusDeliveryStarted})
}

func (s *Server) handleLifecycleUpdate(w http.ResponseWriter, r *http.Request, orderID int64, status string) {
	if status != lifecycle.StatusCanceledBySender {
		if _, err := parseAuthID(r, "X-Courier-ID"); err != nil {
			writeError(w, http.StatusUnauthorized, "missing courier id")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.orders.UpdateStatus(ctx, orderID, status, sql.NullString{}); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		s.logger.Errorf("courier: lifecycle update failed: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

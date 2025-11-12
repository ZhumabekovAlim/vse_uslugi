package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
	"naimuBack/internal/courier/pricing"
	"naimuBack/internal/courier/repo"
	"naimuBack/internal/taxi/timeutil"
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
	ctx = withSenderActor(ctx, senderID)

	dispatchRec := repo.DispatchRecord{RadiusM: s.cfg.GetSearchRadiusStart(), NextTickAt: timeutil.Now(), State: "searching"}

	orderID, err := s.orders.CreateWithDispatch(ctx, order, dispatchRec)
	if err != nil {
		s.logger.Errorf("courier: create order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	if s.dispatcher != nil {
		if err := s.dispatcher.TriggerImmediate(context.Background(), orderID); err != nil {
			s.logger.Errorf("courier: trigger dispatch failed: %v", err)
		}
	}

	resp := map[string]interface{}{
		"order_id":          orderID,
		"recommended_price": recommended,
		"status":            repo.StatusNew,
	}
	writeJSON(w, http.StatusCreated, resp)
	s.emitOrderEvent(ctx, orderID, orderEventTypeCreated, originSender)
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

func (s *Server) handleActiveOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, senderErr := parseAuthID(r, "X-Sender-ID")
	courierID, courierErr := parseAuthID(r, "X-Courier-ID")
	if senderErr != nil && courierErr != nil {
		writeError(w, http.StatusUnauthorized, "missing sender or courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	var (
		order repo.Order
		err   error
	)
	if senderErr == nil {
		order, err = s.orders.ActiveBySender(ctx, senderID)
	} else {
		order, err = s.orders.ActiveByCourier(ctx, courierID)
	}
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "active order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: active order lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load active order")
		return
	}
	writeJSON(w, http.StatusOK, makeOrderResponse(order))
}

func (s *Server) handleCourierOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	limit, offset, err := parsePaging(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	orders, err := s.orders.ListByCourier(ctx, courierID, limit, offset)
	if err != nil {
		s.logger.Errorf("courier: list my orders failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, makeOrderResponse(o))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp})
}

func (s *Server) handleCourierOrdersSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/courier/my/orders/")
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

}

func (s *Server) handleCourierActiveOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	order, err := s.orders.ActiveByCourier(ctx, courierID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "active order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: active courier order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load active order")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"order": makeOrderResponse(order)})
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
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusWaitingFree)
	case "start":
		s.handleStartOrder(w, r, id)
	case "finish":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusCompleted)
	case "confirm-cash":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusClosed)
	case "reprice":
		s.handleReprice(w, r, id)
	case "status":
		s.handleStatus(w, r, id)
	case "review":
		s.handleOrderReview(w, r, id)
	case "waiting":
		if len(parts) == 3 && parts[2] == "advance" {
			s.handleLifecycleWaitingAdvance(w, r, id)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "waypoints":
		if len(parts) == 3 && parts[2] == "next" {
			s.handleLifecycleWaypointNext(w, r, id)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "pause":
		s.handleLifecyclePause(w, r, id)
	case "resume":
		s.handleLifecycleResume(w, r, id)
	case "no-show":
		s.handleLifecycleUpdate(w, r, id, lifecycle.StatusCanceledNoShow)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleReprice(w http.ResponseWriter, r *http.Request, orderID int64) {
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		ClientPrice int `json:"client_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ClientPrice < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("client price must be >= %d", s.cfg.MinPrice))
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	if err := s.orders.UpdatePrice(ctx, orderID, req.ClientPrice); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		s.logger.Errorf("courier: reprice order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update price")
		return
	}
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		s.logger.Errorf("courier: fetch order after reprice failed: %v", err)
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		s.emitOrderEvent(detached, orderID, orderEventTypeUpdated, originSender)
		writeJSON(w, http.StatusOK, map[string]int{"client_price": req.ClientPrice})
		return
	}
	s.emitOrder(ctx, order, orderEventTypeUpdated, originSender)
	writeJSON(w, http.StatusOK, map[string]interface{}{"order": makeOrderResponse(order)})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request, orderID int64) {
	var req struct {
		Status string  `json:"status"`
		Note   *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	status := strings.TrimSpace(strings.ToLower(req.Status))
	if status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load order for status update failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}

	origin := originCourier
	var courierID int64
	var senderID int64
	if status == lifecycle.StatusCanceledBySender {
		var parseErr error
		senderID, parseErr = parseAuthID(r, "X-Sender-ID")
		if parseErr != nil {
			writeError(w, http.StatusUnauthorized, "missing sender id")
			return
		}
		origin = originSender
	} else {
		var parseErr error
		courierID, parseErr = parseAuthID(r, "X-Courier-ID")
		if parseErr != nil {
			writeError(w, http.StatusUnauthorized, "missing courier id")
			return
		}
	}
	ctx = withCourierActor(ctx, courierID)
	ctx = withSenderActor(ctx, senderID)

	if !lifecycle.CanTransition(order.Status, status) {
		writeError(w, http.StatusConflict, "invalid status transition")
		return
	}

	if err := s.orders.UpdateStatusWithNote(ctx, orderID, status, req.Note); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		s.logger.Errorf("courier: update order status failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	// ✅ После обновления — отправляем WebSocket уведомление
	if updated, err := s.orders.Get(ctx, orderID); err == nil {
		s.emitOrder(ctx, updated, orderEventTypeUpdated, origin)
		writeJSON(w, http.StatusOK, map[string]interface{}{"order": makeOrderResponse(updated)})
		return
	} else if err != nil {
		s.logger.Errorf("courier: fetch order after status update failed: %v", err)
	}

	// fallback если не удалось загрузить
	detached := context.WithoutCancel(ctx)
	detached = withCourierActor(detached, courierID)
	detached = withSenderActor(detached, senderID)
	s.emitOrderEvent(detached, orderID, orderEventTypeUpdated, origin)
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (s *Server) handleOrderReview(w http.ResponseWriter, r *http.Request, orderID int64) {
	var payload struct {
		Rating  *float64 `json:"rating"`
		Comment *string  `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}

	ratingProvided := payload.Rating != nil
	commentProvided := payload.Comment != nil

	var ratingValue *float64
	if payload.Rating != nil {
		v := *payload.Rating
		if v < 1 || v > 5 {
			writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
			return
		}
		ratingValue = &v
	}

	var commentValue *string
	if payload.Comment != nil {
		trimmed := strings.TrimSpace(*payload.Comment)
		if trimmed != "" {
			v := trimmed
			commentValue = &v
		} else {
			commentValue = nil
		}
	}

	senderHeader := strings.TrimSpace(r.Header.Get("X-Sender-ID"))
	courierHeader := strings.TrimSpace(r.Header.Get("X-Courier-ID"))

	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	switch {
	case senderHeader != "":
		senderID, err := strconv.ParseInt(senderHeader, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid sender id")
			return
		}
		if !ratingProvided && !commentProvided {
			writeError(w, http.StatusBadRequest, "rating or comment required")
			return
		}
		if err := s.orders.SetSenderReview(ctx, orderID, senderID, ratingValue, commentValue); err != nil {
			switch {
			case errors.Is(err, repo.ErrNotFound):
				writeError(w, http.StatusNotFound, "order not found")
			case errors.Is(err, repo.ErrReviewForbidden):
				writeError(w, http.StatusForbidden, "forbidden")
			case errors.Is(err, repo.ErrReviewOrderNotFinished):
				writeError(w, http.StatusConflict, "order not completed")
			case errors.Is(err, repo.ErrReviewCourierMissing):
				writeError(w, http.StatusConflict, "courier not assigned")
			default:
				s.logger.Errorf("courier: save sender review failed: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to save review")
			}
			return
		}
	case courierHeader != "":
		courierID, err := strconv.ParseInt(courierHeader, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid courier id")
			return
		}
		if commentProvided {
			writeError(w, http.StatusBadRequest, "couriers cannot leave comments")
			return
		}
		if !ratingProvided {
			writeError(w, http.StatusBadRequest, "rating required")
			return
		}
		if err := s.orders.SetCourierReview(ctx, orderID, courierID, ratingValue); err != nil {
			switch {
			case errors.Is(err, repo.ErrNotFound):
				writeError(w, http.StatusNotFound, "order not found")
			case errors.Is(err, repo.ErrReviewForbidden):
				writeError(w, http.StatusForbidden, "forbidden")
			case errors.Is(err, repo.ErrReviewOrderNotFinished):
				writeError(w, http.StatusConflict, "order not completed")
			default:
				s.logger.Errorf("courier: save courier review failed: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to save review")
			}
			return
		}
	default:
		writeError(w, http.StatusUnauthorized, "missing actor id")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLifecycleWaitingAdvance(w http.ResponseWriter, r *http.Request, orderID int64) {
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load order for waiting advance failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}
	if !order.CourierID.Valid || order.CourierID.Int64 != courierID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	next := ""
	switch order.Status {
	case lifecycle.StatusAccepted:
		next = lifecycle.StatusWaitingFree
	case lifecycle.StatusWaitingFree:
		next = lifecycle.StatusInProgress
	default:
		writeError(w, http.StatusConflict, "order not in waiting state")
		return
	}

	if err := s.orders.UpdateStatus(ctx, orderID, next, sql.NullString{}); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		s.logger.Errorf("courier: advance waiting failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to advance order")
		return
	}
	s.emitOrderEvent(ctx, orderID, orderEventTypeUpdated, originCourier)
	writeJSON(w, http.StatusOK, map[string]string{"status": next})
}

func (s *Server) handleLifecycleWaypointNext(w http.ResponseWriter, r *http.Request, orderID int64) {
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load order for waypoint advance failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}
	if !order.CourierID.Valid || order.CourierID.Int64 != courierID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if order.Status != lifecycle.StatusInProgress {
		writeError(w, http.StatusConflict, "order not ready for next waypoint")
		return
	}
	if err := s.orders.UpdateStatus(ctx, orderID, lifecycle.StatusCompleted, sql.NullString{}); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		s.logger.Errorf("courier: advance waypoint failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to advance order")
		return
	}
	s.emitOrderEvent(ctx, orderID, orderEventTypeUpdated, originCourier)
	writeJSON(w, http.StatusOK, map[string]string{"status": lifecycle.StatusCompleted})
}

func (s *Server) handleLifecyclePause(w http.ResponseWriter, r *http.Request, orderID int64) {
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load order for pause failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}
	if !order.CourierID.Valid || order.CourierID.Int64 != courierID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleResume(w http.ResponseWriter, r *http.Request, orderID int64) {
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	order, err := s.orders.Get(ctx, orderID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load order for resume failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load order")
		return
	}
	if !order.CourierID.Valid || order.CourierID.Int64 != courierID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
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
	origin := originUnknown
	var courierID int64
	var senderID int64
	if id, err := parseAuthID(r, "X-Sender-ID"); err == nil {
		actorStatus = lifecycle.StatusCanceledBySender
		origin = originSender
		senderID = id
	} else if id, err := parseAuthID(r, "X-Courier-ID"); err == nil {
		actorStatus = lifecycle.StatusCanceledByCourier
		origin = originCourier
		courierID = id
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
	ctx = withCourierActor(ctx, courierID)
	ctx = withSenderActor(ctx, senderID)

	if err := s.orders.UpdateStatusWithNote(ctx, orderID, actorStatus, req.Reason); errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	} else if err != nil {
		s.logger.Errorf("courier: cancel order failed: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": actorStatus})
	s.emitOrderEvent(ctx, orderID, orderEventTypeUpdated, origin)
}

func (s *Server) handleStartOrder(w http.ResponseWriter, r *http.Request, orderID int64) {
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)

	if err := s.orders.UpdateStatus(ctx, orderID, lifecycle.StatusInProgress, sql.NullString{}); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		s.logger.Errorf("courier: update order status %s failed: %v", lifecycle.StatusInProgress, err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": lifecycle.StatusInProgress})
	s.emitOrderEvent(ctx, orderID, orderEventTypeUpdated, originCourier)
}

func (s *Server) handleLifecycleUpdate(w http.ResponseWriter, r *http.Request, orderID int64, status string) {
	origin := originCourier
	var courierID int64
	var senderID int64
	if status != lifecycle.StatusCanceledBySender {
		var err error
		courierID, err = parseAuthID(r, "X-Courier-ID")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing courier id")
			return
		}
	} else {
		var err error
		senderID, err = parseAuthID(r, "X-Sender-ID")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing sender id")
			return
		}
		origin = originSender
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)
	ctx = withSenderActor(ctx, senderID)

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
	s.emitOrderEvent(ctx, orderID, orderEventTypeUpdated, origin)
}

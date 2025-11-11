package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
	"naimuBack/internal/courier/repo"
)

func (s *Server) handleOfferPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	var req struct {
		OrderID int64 `json:"order_id"`
		Price   int   `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}
	if req.Price < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("price must be >= %d", s.cfg.MinPrice))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)

	if err := s.offers.Upsert(ctx, req.OrderID, courierID, req.Price); err != nil {
		s.logger.Errorf("courier: upsert offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to store offer")
		return
	}

	price := req.Price
	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		s.logger.Errorf("courier: load order %d after offer price failed: %v", req.OrderID, err)
		detached := withCourierActor(context.WithoutCancel(ctx), courierID)
		s.emitOfferEvent(detached, req.OrderID, courierID, repo.OfferStatusProposed, &price, originCourier)
		s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originCourier)
	} else {
		s.emitOffer(ctx, order, courierID, repo.OfferStatusProposed, &price, originCourier)
		s.emitOrder(ctx, order, orderEventTypeUpdated, originCourier)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusNew})
}

func (s *Server) handleOfferAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64 `json:"order_id"`
		CourierID int64 `json:"courier_id"`
		Price     int   `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}
	if req.Price < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("price must be >= %d", s.cfg.MinPrice))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusAccepted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		s.logger.Errorf("courier: accept offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to accept offer")
		return
	}

	if err := s.orders.AssignCourier(ctx, req.OrderID, req.CourierID, lifecycle.StatusAccepted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		s.logger.Errorf("courier: assign courier failed: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	price := req.Price
	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		s.logger.Errorf("courier: load order %d after offer accept failed: %v", req.OrderID, err)
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusAccepted, &price, originSender)
		s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originSender)
	} else {
		s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusAccepted, &price, originSender)
		s.emitOrder(ctx, order, orderEventTypeUpdated, originSender)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusAccepted})
}

func (s *Server) handleOfferDecline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64 `json:"order_id"`
		CourierID int64 `json:"courier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusDeclined); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		s.logger.Errorf("courier: decline offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to decline offer")
		return
	}

	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		s.logger.Errorf("courier: load order %d after offer decline failed: %v", req.OrderID, err)
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
	} else {
		s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.OfferStatusDeclined})
}

func (s *Server) handleOfferRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64  `json:"order_id"`
		CourierID int64  `json:"courier_id"`
		Decision  string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != "accept" && decision != "decline" {
		writeError(w, http.StatusBadRequest, "decision must be accept or decline")
		return
	}

	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	if decision == "accept" {
		if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusAccepted); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "offer not found")
				return
			}
			s.logger.Errorf("courier: respond offer accept failed: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update offer")
			return
		}
		if err := s.orders.AssignCourier(ctx, req.OrderID, req.CourierID, lifecycle.StatusAccepted); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "order not found")
				return
			}
			s.logger.Errorf("courier: assign courier on respond failed: %v", err)
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusDeclined); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "offer not found")
				return
			}
			s.logger.Errorf("courier: respond offer decline failed: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to update offer")
			return
		}
	}

	if order, err := s.orders.Get(ctx, req.OrderID); err == nil {
		status := repo.OfferStatusDeclined
		if decision == "accept" {
			status = repo.OfferStatusAccepted
			s.emitOrder(ctx, order, orderEventTypeUpdated, originSender)
		}
		s.emitOffer(ctx, order, req.CourierID, status, nil, originSender)
	} else {
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		status := repo.OfferStatusDeclined
		if decision == "accept" {
			status = repo.OfferStatusAccepted
			s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originSender)
		}
		s.emitOfferEvent(detached, req.OrderID, req.CourierID, status, nil, originSender)
	}

	respStatus := map[string]string{"status": "declined"}
	if decision == "accept" {
		respStatus["status"] = string(repo.StatusAccepted)
	}
	writeJSON(w, http.StatusOK, respStatus)
}

package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	if err := s.offers.Upsert(ctx, req.OrderID, courierID, req.Price); err != nil {
		s.logger.Errorf("courier: upsert offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to store offer")
		return
	}
	if err := s.orders.UpdateStatus(ctx, req.OrderID, lifecycle.StatusOffered, sql.NullString{}); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		// ignore transition conflict when order already advanced
		s.logger.Infof("courier: skipping status update after offer: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusOffered})
}

func (s *Server) handleOfferAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, err := parseAuthID(r, "X-Sender-ID"); err != nil {
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

	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusAccepted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		s.logger.Errorf("courier: accept offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to accept offer")
		return
	}

	if err := s.orders.AssignCourier(ctx, req.OrderID, req.CourierID, lifecycle.StatusAssigned); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		s.logger.Errorf("courier: assign courier failed: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusAssigned})
}

func (s *Server) handleOfferDecline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, err := parseAuthID(r, "X-Sender-ID"); err != nil {
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

	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusDeclined); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		s.logger.Errorf("courier: decline offer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to decline offer")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.OfferStatusDeclined})
}

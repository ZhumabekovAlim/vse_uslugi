package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"naimuBack/internal/courier/repo"
)

func (s *Server) handleAdminCourierOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.orders == nil {
		http.Error(w, "orders repository unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, offset, err := parsePaging(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	orders, err := s.orders.ListAll(ctx, limit, offset)
	if err != nil {
		s.logger.Errorf("courier: admin list orders failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		resp = append(resp, makeOrderResponse(order))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp})
}

func (s *Server) handleAdminCourierOrdersStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.orders == nil {
		http.Error(w, "orders repository unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	stats, err := s.orders.Stats(ctx)
	if err != nil {
		s.logger.Errorf("courier: admin orders stats failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAdminCouriers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, offset, err := parsePaging(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	couriers, err := s.couriers.List(ctx, limit, offset)
	if err != nil {
		s.logger.Errorf("courier: admin list couriers failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list couriers")
		return
	}
	resp := make([]courierResponse, 0, len(couriers))
	for _, c := range couriers {
		resp = append(resp, makeCourierResponse(c))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"couriers": resp})
}

func (s *Server) handleAdminCouriersStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	stats, err := s.couriers.Stats(ctx)
	if err != nil {
		s.logger.Errorf("courier: admin couriers stats failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAdminCourierActions(w http.ResponseWriter, r *http.Request) {
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/courier/couriers/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid courier id")
		return
	}
	switch parts[1] {
	case "ban":
		s.handleAdminCourierBan(w, r, id)
	case "approval":
		s.handleAdminCourierApproval(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleAdminCourierBan(w http.ResponseWriter, r *http.Request, courierID int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Ban bool `json:"ban"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	status := "banned"
	if !payload.Ban {
		status = "active"
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	if err := s.couriers.UpdateStatus(ctx, courierID, status); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "courier not found")
			return
		}
		s.logger.Errorf("courier: admin ban failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update courier")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (s *Server) handleAdminCourierApproval(w http.ResponseWriter, r *http.Request, courierID int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	status := strings.TrimSpace(strings.ToLower(payload.Status))
	if status == "" {
		status = "approved"
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	if err := s.couriers.UpdateStatus(ctx, courierID, status); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "courier not found")
			return
		}
		s.logger.Errorf("courier: admin approval failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update courier")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

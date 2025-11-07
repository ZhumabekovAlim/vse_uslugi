package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type TaxiIntercityOrderHandler struct {
	Service *services.TaxiIntercityOrderService
}

var allowedTripTypes = map[string]struct{}{
	"with_companions":    {},
	"parcel":             {},
	"without_companions": {},
}

func (h *TaxiIntercityOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaxiIntercityOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	clientID, ok := r.Context().Value("user_id").(int)
	if !ok || clientID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	req.FromCity = strings.TrimSpace(req.FromCity)
	req.ToCity = strings.TrimSpace(req.ToCity)
	req.TripType = strings.ToLower(strings.TrimSpace(req.TripType))
	req.Comment = strings.TrimSpace(req.Comment)
	req.DepartureDate = strings.TrimSpace(req.DepartureDate)

	if req.FromCity == "" || req.ToCity == "" || req.DepartureDate == "" || req.TripType == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	if _, ok := allowedTripTypes[req.TripType]; !ok {
		http.Error(w, "Invalid trip_type", http.StatusBadRequest)
		return
	}
	if req.Price <= 0 {
		http.Error(w, "Price must be greater than zero", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", req.DepartureDate); err != nil {
		http.Error(w, "Invalid departure_date, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	order, err := h.Service.Create(r.Context(), clientID, req)
	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *TaxiIntercityOrderHandler) Search(w http.ResponseWriter, r *http.Request) {
	filter := models.TaxiIntercityOrderFilter{}
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		filter.FromCity = v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		filter.ToCity = v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("date")); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DepartureDate = &t
		} else {
			http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		filter.Status = v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if limit, err := strconv.Atoi(v); err == nil {
			filter.Limit = limit
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if offset, err := strconv.Atoi(v); err == nil {
			filter.Offset = offset
		}
	}

	orders, err := h.Service.Search(r.Context(), filter)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(struct {
		Orders []models.TaxiIntercityOrder `json:"orders"`
	}{Orders: orders})
}

func (h *TaxiIntercityOrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrTaxiIntercityOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch order", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(order)
}

func (h *TaxiIntercityOrderHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	clientID, ok := r.Context().Value("user_id").(int)
	if !ok || clientID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))

	orders, err := h.Service.ListByClient(r.Context(), clientID, status)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(struct {
		Orders []models.TaxiIntercityOrder `json:"orders"`
	}{Orders: orders})
}

func (h *TaxiIntercityOrderHandler) Close(w http.ResponseWriter, r *http.Request) {
	clientID, ok := r.Context().Value("user_id").(int)
	if !ok || clientID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid order id", http.StatusBadRequest)
		return
	}

	if err := h.Service.Close(r.Context(), id, clientID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrTaxiIntercityOrderNotFound):
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrTaxiIntercityOrderForbidden):
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		default:
			http.Error(w, "Failed to close order", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

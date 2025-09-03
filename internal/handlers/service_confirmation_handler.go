package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type ServiceConfirmationHandler struct {
	Service *services.ServiceConfirmationService
}

func (h *ServiceConfirmationHandler) ConfirmService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID   int `json:"service_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmService(r.Context(), req.ServiceID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm service", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ServiceConfirmationHandler) CancelService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID int `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.Service.CancelService(r.Context(), req.ServiceID, userID); err != nil {
		if errors.Is(err, repositories.ErrServiceConfirmationNotFound) {
			http.Error(w, "Service confirmation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Could not cancel service", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ServiceConfirmationHandler) DoneService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID int `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.DoneService(r.Context(), req.ServiceID); err != nil {
		http.Error(w, "Could not mark service done", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

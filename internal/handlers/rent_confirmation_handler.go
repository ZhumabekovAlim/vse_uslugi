package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

type RentConfirmationHandler struct {
	Service *services.RentConfirmationService
}

func (h *RentConfirmationHandler) ConfirmRent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentID   int `json:"rent_id"`
		ClientID int `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmRent(r.Context(), req.RentID, req.ClientID); err != nil {
		http.Error(w, "Could not confirm rent", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentConfirmationHandler) CancelRent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentID int `json:"rent_id"`
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
	if err := h.Service.CancelRent(r.Context(), req.RentID, userID); err != nil {
		http.Error(w, "Could not cancel rent", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentConfirmationHandler) DoneRent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentID int `json:"rent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.DoneRent(r.Context(), req.RentID); err != nil {
		http.Error(w, "Could not mark rent done", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

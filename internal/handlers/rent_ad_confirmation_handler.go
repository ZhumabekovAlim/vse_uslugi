package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

type RentAdConfirmationHandler struct {
	Service *services.RentAdConfirmationService
}

func (h *RentAdConfirmationHandler) ConfirmRentAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentAdID    int `json:"rent_ad_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmRentAd(r.Context(), req.RentAdID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm rent ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentAdConfirmationHandler) CancelRentAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentAdID int `json:"rent_ad_id"`
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
	if err := h.Service.CancelRentAd(r.Context(), req.RentAdID, userID); err != nil {
		http.Error(w, "Could not cancel rent ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentAdConfirmationHandler) DoneRentAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentAdID int `json:"rent_ad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.DoneRentAd(r.Context(), req.RentAdID); err != nil {
		http.Error(w, "Could not mark rent ad done", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

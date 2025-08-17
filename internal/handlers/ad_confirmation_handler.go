package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

type AdConfirmationHandler struct {
	Service *services.AdConfirmationService
}

func (h *AdConfirmationHandler) ConfirmAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID        int `json:"ad_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmAd(r.Context(), req.AdID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AdConfirmationHandler) CancelAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID int `json:"ad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.CancelAd(r.Context(), req.AdID); err != nil {
		http.Error(w, "Could not cancel ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

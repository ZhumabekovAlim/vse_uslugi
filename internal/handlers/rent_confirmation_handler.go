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
		RentID      int `json:"rent_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmRent(r.Context(), req.RentID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm rent", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

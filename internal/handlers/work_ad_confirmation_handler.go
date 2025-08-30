package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

type WorkAdConfirmationHandler struct {
	Service *services.WorkAdConfirmationService
}

func (h *WorkAdConfirmationHandler) ConfirmWorkAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkAdID    int `json:"work_ad_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmWorkAd(r.Context(), req.WorkAdID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm work ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WorkAdConfirmationHandler) CancelWorkAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkAdID int `json:"work_ad_id"`
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
	if err := h.Service.CancelWorkAd(r.Context(), req.WorkAdID, userID); err != nil {
		http.Error(w, "Could not cancel work ad", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WorkAdConfirmationHandler) DoneWorkAd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkAdID int `json:"work_ad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.DoneWorkAd(r.Context(), req.WorkAdID); err != nil {
		http.Error(w, "Could not mark work ad done", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

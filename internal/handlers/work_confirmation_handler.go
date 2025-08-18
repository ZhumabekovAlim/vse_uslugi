package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

type WorkConfirmationHandler struct {
	Service *services.WorkConfirmationService
}

func (h *WorkConfirmationHandler) ConfirmWork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkID      int `json:"work_id"`
		PerformerID int `json:"performer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ConfirmWork(r.Context(), req.WorkID, req.PerformerID); err != nil {
		http.Error(w, "Could not confirm work", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WorkConfirmationHandler) CancelWork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkID int `json:"work_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.CancelWork(r.Context(), req.WorkID); err != nil {
		http.Error(w, "Could not cancel work", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WorkConfirmationHandler) DoneWork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkID int `json:"work_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.DoneWork(r.Context(), req.WorkID); err != nil {
		http.Error(w, "Could not mark work done", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

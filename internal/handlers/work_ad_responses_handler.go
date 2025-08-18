package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type WorkAdResponseHandler struct {
	Service *services.WorkAdResponseService
}

func (h *WorkAdResponseHandler) CreateWorkAdResponse(w http.ResponseWriter, r *http.Request) {
	var input models.WorkAdResponses

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.CreateWorkAdResponse(r.Context(), input)
	if err != nil {
		if errors.Is(err, models.ErrAlreadyResponded) {

			http.Error(w, "already responded", http.StatusOK)

			return
		}
		http.Error(w, "Could not create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

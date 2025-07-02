package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type AdResponseHandler struct {
	Service *services.AdResponseService
}

func (h *AdResponseHandler) CreateAdResponse(w http.ResponseWriter, r *http.Request) {
	var input models.AdResponses

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.CreateAdResponse(r.Context(), input)
	if err != nil {
		http.Error(w, "Could not create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

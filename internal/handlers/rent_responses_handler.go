package handlers

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type RentResponseHandler struct {
	Service *services.RentResponseService
}

func (h *RentResponseHandler) CreateRentResponse(w http.ResponseWriter, r *http.Request) {
	var input models.RentResponses

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.CreateRentResponse(r.Context(), input)
	if err != nil {
		http.Error(w, "Could not create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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
		if errors.Is(err, models.ErrAlreadyResponded) {
			http.Error(w, "already responded", http.StatusOK)
			return
		}
		if errors.Is(err, models.ErrNoRemainingResponses) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "Could not create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AdResponseHandler) CancelAdResponse(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	responseID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid response id", http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.Service.CancelAdResponse(r.Context(), responseID, userID); err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
			http.Error(w, "response not found", http.StatusNotFound)
		case errors.Is(err, models.ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		default:
			http.Error(w, "could not cancel response", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

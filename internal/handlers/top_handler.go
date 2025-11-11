package handlers

import (
	"encoding/json"
	"errors"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
	"net/http"
)

type TopHandler struct {
	Service *services.TopService
}

type topResponse struct {
	Top models.TopInfo `json:"top"`
	Raw string         `json:"raw"`
}

func (h *TopHandler) Activate(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Service == nil {
		http.Error(w, "top service not configured", http.StatusInternalServerError)
		return
	}

	userID, ok := r.Context().Value("user_id").(int)
	if !ok || userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.TopActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	info, err := h.Service.ActivateTop(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidTopType),
			errors.Is(err, models.ErrInvalidTopDuration),
			errors.Is(err, models.ErrInvalidTopID):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, repositories.ErrListingNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, services.ErrTopForbidden):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	raw, err := info.Marshal()
	if err != nil {
		http.Error(w, "failed to encode top payload", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(topResponse{Top: info, Raw: raw}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

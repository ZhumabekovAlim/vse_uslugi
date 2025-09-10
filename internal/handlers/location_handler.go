package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

// LocationHandler provides HTTP endpoints for user locations.
type LocationHandler struct {
	Service   *services.LocationService
	Broadcast chan<- models.Location
}

// UpdateLocation stores coordinates and broadcasts to listeners.
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var loc models.Location
	if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetLocation(r.Context(), loc); err != nil {
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}
	if h.Broadcast != nil {
		h.Broadcast <- loc
	}
	w.WriteHeader(http.StatusOK)
}

// GetLocation returns last known coordinates for a user.
func (h *LocationHandler) GetLocation(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}
	loc, err := h.Service.GetLocation(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get location", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(loc)
}

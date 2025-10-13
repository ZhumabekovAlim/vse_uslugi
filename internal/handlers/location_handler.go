package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

// GoOffline clears location for a user and marks them offline.
func (h *LocationHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.GoOffline(r.Context(), payload.UserID); err != nil {
		http.Error(w, "Failed to go offline", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetExecutors returns online executors with active items filtered by request body.
func (h *LocationHandler) GetExecutors(w http.ResponseWriter, r *http.Request) {
	var filter models.ExecutorLocationFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if rawType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(":type"))); rawType != "" {
		normalized := strings.ReplaceAll(rawType, "-", "_")
		switch normalized {
		case "service", "work", "rent", "ad", "work_ad", "rent_ad":
			filter.Type = normalized
		default:
			http.Error(w, "Invalid executor type", http.StatusBadRequest)
			return
		}
	}

	execs, err := h.Service.GetExecutors(r.Context(), filter)
	if err != nil {
		http.Error(w, "Failed to get executors", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(execs)
}

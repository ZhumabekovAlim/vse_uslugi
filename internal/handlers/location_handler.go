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
	Service  *services.LocationService
	Notifier LocationNotifier
}

// LocationNotifier describes callbacks used to fan out location updates over websockets.
type LocationNotifier interface {
	BroadcastLocation(models.Location)
	NotifyBusinessWorkerOffline(workerUserID, businessUserID int, marker *models.BusinessAggregatedMarker)
}

// UpdateLocation stores coordinates and broadcasts to listeners.
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var loc models.Location
	if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if loc.UserID == 0 {
		loc.UserID, _ = r.Context().Value("user_id").(int)
	}
	if loc.UserID == 0 {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetLocation(r.Context(), loc); err != nil {
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}
	if h.Notifier != nil {
		h.Notifier.BroadcastLocation(loc)
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
		UserID    int      `json:"user_id"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if payload.UserID == 0 {
		payload.UserID, _ = r.Context().Value("user_id").(int)
	}
	if payload.UserID == 0 {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	role, _ := r.Context().Value("role").(string)

	if role == "business_worker" {
		businessUserID, marker, err := h.Service.SetBusinessWorkerOffline(r.Context(), payload.UserID)
		if err != nil {
			http.Error(w, "Failed to go offline", http.StatusInternalServerError)
			return
		}
		if h.Notifier != nil {
			h.Notifier.NotifyBusinessWorkerOffline(payload.UserID, businessUserID, marker)
		}
	} else {
		if err := h.Service.GoOffline(r.Context(), payload.UserID); err != nil {
			http.Error(w, "Failed to go offline", http.StatusInternalServerError)
			return
		}
		if h.Notifier != nil {
			h.Notifier.BroadcastLocation(models.Location{UserID: payload.UserID})
		}
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

// GetBusinessWorkers returns worker coordinates for the authenticated business account.
func (h *LocationHandler) GetBusinessWorkers(w http.ResponseWriter, r *http.Request) {
	businessUserID, _ := r.Context().Value("user_id").(int)
	if businessUserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var filter models.ExecutorLocationFilter
	_ = json.NewDecoder(r.Body).Decode(&filter)
	filter.BusinessUserID = businessUserID

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

	_ = json.NewEncoder(w).Encode(map[string]any{"workers": execs})
}

// GetBusinessMarkers returns aggregated markers for all businesses with online workers.
func (h *LocationHandler) GetBusinessMarkers(w http.ResponseWriter, r *http.Request) {
	markers, err := h.Service.GetBusinessMarkers(r.Context())
	if err != nil {
		http.Error(w, "Failed to load business markers", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"markers": markers})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/services"
)

type SubscriptionHandler struct {
	Service *services.SubscriptionService
}

// GetSubscription returns subscription info for specified user.
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	profile, err := h.Service.GetProfile(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

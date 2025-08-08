package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/services"
)

// UserResponsesHandler handles HTTP requests for user responses.
type UserResponsesHandler struct {
	Service *services.UserResponsesService
}

// GetResponsesByUserID returns all responses for the specified user.
func (h *UserResponsesHandler) GetResponsesByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	responses, err := h.Service.GetResponsesByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get responses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

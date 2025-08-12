package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/services"
)

// UserReviewsHandler handles HTTP requests for user reviews.
type UserReviewsHandler struct {
	Service *services.UserReviewsService
}

// GetReviewsByUserID returns all reviews for the specified user.
func (h *UserReviewsHandler) GetReviewsByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	reviews, err := h.Service.GetReviewsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get reviews", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

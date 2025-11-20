package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type ReviewHandler struct {
	Service *services.ReviewService
}

func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var reviews models.Reviews
	if err := json.NewDecoder(r.Body).Decode(&reviews); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.CreateReview(r.Context(), reviews)
	if err != nil {
		if errors.Is(err, models.ErrAlreadyReviewed) {
			http.Error(w, "user already reviewed", http.StatusConflict)
			return
		}
		log.Printf("CreateReview error: %v", err)
		http.Error(w, "Failed to create review", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *ReviewHandler) GetReviewsByServiceID(w http.ResponseWriter, r *http.Request) {
	serviceIDStr := r.URL.Query().Get(":service_id")
	serviceID, err := strconv.Atoi(serviceIDStr)
	if err != nil {
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.GetReviewsByServiceID(r.Context(), serviceID)
	if err != nil {
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil || reviewID == 0 {
		http.Error(w, "Invalid or missing review ID", http.StatusBadRequest)
		return
	}

	var review models.Reviews
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	review.ID = reviewID
	if err := h.Service.UpdateReview(r.Context(), review); err != nil {
		http.Error(w, "Failed to update review", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	if err := h.Service.DeleteReview(r.Context(), reviewID); err != nil {
		http.Error(w, "Failed to delete review", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

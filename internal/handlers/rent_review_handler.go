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

type RentReviewHandler struct {
	Service *services.RentReviewService
}

func (h *RentReviewHandler) CreateRentReview(w http.ResponseWriter, r *http.Request) {
	var reviews models.RentReviews
	if err := json.NewDecoder(r.Body).Decode(&reviews); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.CreateRentReview(r.Context(), reviews)
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

func (h *RentReviewHandler) GetRentReviewsByRentID(w http.ResponseWriter, r *http.Request) {
	rentIDStr := r.URL.Query().Get(":rent_id")
	rentID, err := strconv.Atoi(rentIDStr)
	if err != nil {
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.GetRentReviewsByRentID(r.Context(), rentID)
	if err != nil {
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *RentReviewHandler) UpdateRentReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil || reviewID == 0 {
		http.Error(w, "Invalid or missing review ID", http.StatusBadRequest)
		return
	}

	var review models.RentReviews
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	review.ID = reviewID
	if err := h.Service.UpdateRentReview(r.Context(), review); err != nil {
		http.Error(w, "Failed to update review", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *RentReviewHandler) DeleteRentReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	if err := h.Service.DeleteRentReview(r.Context(), reviewID); err != nil {
		http.Error(w, "Failed to delete review", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

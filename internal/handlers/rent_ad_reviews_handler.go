package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type RentAdReviewHandler struct {
	Service *services.RentAdReviewService
}

func (h *RentAdReviewHandler) CreateRentAdReview(w http.ResponseWriter, r *http.Request) {
	var reviews models.RentAdReviews
	if err := json.NewDecoder(r.Body).Decode(&reviews); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.CreateRentAdReview(r.Context(), reviews)
	if err != nil {
		log.Printf("CreateReview error: %v", err)
		http.Error(w, "Failed to create review", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *RentAdReviewHandler) GetRentAdReviewsByRentID(w http.ResponseWriter, r *http.Request) {
	rentAdIDStr := r.URL.Query().Get(":rent_ad_id")
	rentAdID, err := strconv.Atoi(rentAdIDStr)
	if err != nil {
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.GetRentAdReviewsByRentID(r.Context(), rentAdID)
	if err != nil {
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *RentAdReviewHandler) UpdateRentAdReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil || reviewID == 0 {
		http.Error(w, "Invalid or missing review ID", http.StatusBadRequest)
		return
	}

	var review models.RentAdReviews
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	review.ID = reviewID
	if err := h.Service.UpdateRentAdReview(r.Context(), review); err != nil {
		http.Error(w, "Failed to update review", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *RentAdReviewHandler) DeleteRentAdReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	if err := h.Service.DeleteRentAdReview(r.Context(), reviewID); err != nil {
		http.Error(w, "Failed to delete review", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type WorkAdReviewHandler struct {
	Service *services.WorkAdReviewService
}

func (h *WorkAdReviewHandler) CreateWorkAdReview(w http.ResponseWriter, r *http.Request) {
	var reviews models.WorkAdReviews
	if err := json.NewDecoder(r.Body).Decode(&reviews); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.CreateWorkAdReview(r.Context(), reviews)
	if err != nil {
		log.Printf("CreateReview error: %v", err)
		http.Error(w, "Failed to create review", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *WorkAdReviewHandler) GetWorkAdReviewsByWorkID(w http.ResponseWriter, r *http.Request) {
	workAdIDStr := r.URL.Query().Get(":workad_id")
	workAdID, err := strconv.Atoi(workAdIDStr)
	if err != nil {
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
		return
	}
	reviews, err := h.Service.GetWorkAdReviewsByWorkID(r.Context(), workAdID)
	if err != nil {
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reviews)
}

func (h *WorkAdReviewHandler) UpdateWorkAdReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil || reviewID == 0 {
		http.Error(w, "Invalid or missing review ID", http.StatusBadRequest)
		return
	}

	var review models.WorkAdReviews
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	review.ID = reviewID
	if err := h.Service.UpdateWorkAdReview(r.Context(), review); err != nil {
		http.Error(w, "Failed to update review", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WorkAdReviewHandler) DeleteWorkAdReview(w http.ResponseWriter, r *http.Request) {
	reviewIDStr := r.URL.Query().Get(":id")
	reviewID, err := strconv.Atoi(reviewIDStr)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	if err := h.Service.DeleteWorkAdReview(r.Context(), reviewID); err != nil {
		http.Error(w, "Failed to delete review", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/services"
)

// UserItemsHandler handles HTTP requests for user items.
type UserItemsHandler struct {
	Service *services.UserItemsService
}

// GetPostsByUserID returns all services, works and rents for the specified user.
func (h *UserItemsHandler) GetPostsByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	items, err := h.Service.GetServiceWorkRentByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetAdsByUserID returns all ads, work_ads and rent_ads for the specified user.
func (h *UserItemsHandler) GetAdsByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	items, err := h.Service.GetAdWorkAdRentAdByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetOrderHistoryByUserID returns all completed services, works, rents and ads for the specified user.
func (h *UserItemsHandler) GetOrderHistoryByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	items, err := h.Service.GetOrderHistoryByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetActiveOrdersByUserID returns all active orders where the user is performer.
func (h *UserItemsHandler) GetActiveOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	items, err := h.Service.GetActiveOrdersByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type RentAdFavoriteHandler struct {
	Service *services.RentAdFavoriteService
}

func (h *RentAdFavoriteHandler) AddRentAdToFavorites(w http.ResponseWriter, r *http.Request) {
	var fav models.RentAdFavorite
	if err := json.NewDecoder(r.Body).Decode(&fav); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.AddRentAdToFavorites(r.Context(), fav); err != nil {
		http.Error(w, "Failed to add to favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *RentAdFavoriteHandler) RemoveRentAdFromFavorites(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	rentAdIDStr := r.URL.Query().Get(":rent_ad_id")

	userID, err1 := strconv.Atoi(userIDStr)
	rentAdID, err2 := strconv.Atoi(rentAdIDStr)
	if err1 != nil || err2 != nil {
		log.Printf("RemoveFavorite error: %v", err1)
		log.Printf("RemoveFavorite error: %v", err2)
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	if err := h.Service.RemoveRentAdFromFavorites(r.Context(), userID, rentAdID); err != nil {
		http.Error(w, "Failed to remove from favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentAdFavoriteHandler) IsRentAdFavorite(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	rentAdIDStr := r.URL.Query().Get(":rent_ad_id")

	userID, err1 := strconv.Atoi(userIDStr)
	rentAdID, err2 := strconv.Atoi(rentAdIDStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	liked, err := h.Service.IsRentAdFavorite(r.Context(), userID, rentAdID)
	if err != nil {
		http.Error(w, "Failed to check favorite status", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"liked": liked})
}

func (h *RentAdFavoriteHandler) GetRentAdFavoritesByUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	favs, err := h.Service.GetRentAdFavoritesByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get favorites", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(favs)
}

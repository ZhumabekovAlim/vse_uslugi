package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type AdFavoriteHandler struct {
	Service *services.AdFavoriteService
}

func (h *AdFavoriteHandler) AddAdToFavorites(w http.ResponseWriter, r *http.Request) {
	var fav models.AdFavorite
	if err := json.NewDecoder(r.Body).Decode(&fav); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.AddAdToFavorites(r.Context(), fav); err != nil {
		http.Error(w, "Failed to add to favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *AdFavoriteHandler) RemoveAdFromFavorites(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	adIDStr := r.URL.Query().Get(":ad_id")

	userID, err1 := strconv.Atoi(userIDStr)
	adID, err2 := strconv.Atoi(adIDStr)
	if err1 != nil || err2 != nil {
		log.Printf("RemoveFavorite error: %v", err1)
		log.Printf("RemoveFavorite error: %v", err2)
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	if err := h.Service.RemoveAdFromFavorites(r.Context(), userID, adID); err != nil {
		http.Error(w, "Failed to remove from favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AdFavoriteHandler) IsAdFavorite(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	adIDStr := r.URL.Query().Get(":ad_id")

	userID, err1 := strconv.Atoi(userIDStr)
	adID, err2 := strconv.Atoi(adIDStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	liked, err := h.Service.IsAdFavorite(r.Context(), userID, adID)
	if err != nil {
		http.Error(w, "Failed to check favorite status", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"liked": liked})
}

func (h *AdFavoriteHandler) GetAdFavoritesByUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	favs, err := h.Service.GetAdFavoritesByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get favorites", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(favs)
}

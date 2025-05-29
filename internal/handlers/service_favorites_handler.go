package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type ServiceFavoriteHandler struct {
	Service *services.ServiceFavoriteService
}

func (h *ServiceFavoriteHandler) AddToFavorites(w http.ResponseWriter, r *http.Request) {
	var fav models.ServiceFavorite
	if err := json.NewDecoder(r.Body).Decode(&fav); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.AddToFavorites(r.Context(), fav); err != nil {
		http.Error(w, "Failed to add to favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *ServiceFavoriteHandler) RemoveFromFavorites(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	serviceIDStr := r.URL.Query().Get(":service_id")

	userID, err1 := strconv.Atoi(userIDStr)
	serviceID, err2 := strconv.Atoi(serviceIDStr)
	if err1 != nil || err2 != nil {
		log.Printf("RemoveFavorite error: %v", err1)
		log.Printf("RemoveFavorite error: %v", err2)
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	if err := h.Service.RemoveFromFavorites(r.Context(), userID, serviceID); err != nil {
		http.Error(w, "Failed to remove from favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ServiceFavoriteHandler) IsFavorite(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	serviceIDStr := r.URL.Query().Get(":service_id")

	userID, err1 := strconv.Atoi(userIDStr)
	serviceID, err2 := strconv.Atoi(serviceIDStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	liked, err := h.Service.IsFavorite(r.Context(), userID, serviceID)
	if err != nil {
		http.Error(w, "Failed to check favorite status", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"liked": liked})
}

func (h *ServiceFavoriteHandler) GetFavoritesByUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	favs, err := h.Service.GetFavoritesByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get favorites", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(favs)
}

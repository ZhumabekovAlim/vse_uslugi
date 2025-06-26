package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type WorkFavoriteHandler struct {
	Service *services.WorkFavoriteService
}

func (h *WorkFavoriteHandler) AddWorkToFavorites(w http.ResponseWriter, r *http.Request) {
	var fav models.WorkFavorite
	if err := json.NewDecoder(r.Body).Decode(&fav); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.AddWorkToFavorites(r.Context(), fav); err != nil {
		http.Error(w, "Failed to add to favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WorkFavoriteHandler) RemoveWorkFromFavorites(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	workIDStr := r.URL.Query().Get(":work_id")

	userID, err1 := strconv.Atoi(userIDStr)
	workID, err2 := strconv.Atoi(workIDStr)
	if err1 != nil || err2 != nil {
		log.Printf("RemoveFavorite error: %v", err1)
		log.Printf("RemoveFavorite error: %v", err2)
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	if err := h.Service.RemoveWorkFromFavorites(r.Context(), userID, workID); err != nil {
		http.Error(w, "Failed to remove from favorites", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WorkFavoriteHandler) IsWorkFavorite(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	workIDStr := r.URL.Query().Get(":work_id")

	userID, err1 := strconv.Atoi(userIDStr)
	workID, err2 := strconv.Atoi(workIDStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid user_id or service_id", http.StatusBadRequest)
		return
	}

	liked, err := h.Service.IsWorkFavorite(r.Context(), userID, workID)
	if err != nil {
		http.Error(w, "Failed to check favorite status", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"liked": liked})
}

func (h *WorkFavoriteHandler) GetWorkFavoritesByUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	favs, err := h.Service.GetWorkFavoritesByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get favorites", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(favs)
}

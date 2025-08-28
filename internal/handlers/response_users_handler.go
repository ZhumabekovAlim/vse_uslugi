package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/services"
)

// ResponseUsersHandler handles HTTP requests for retrieving users who responded to items.
type ResponseUsersHandler struct {
	Service *services.ResponseUsersService
}

// GetUsersByItemID returns users who responded to a specific item.
func (h *ResponseUsersHandler) GetUsersByItemID(w http.ResponseWriter, r *http.Request) {
	itemType := r.URL.Query().Get(":type")
	idStr := r.URL.Query().Get(":item_id")
	itemID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid item_id", http.StatusBadRequest)
		return
	}

	users, err := h.Service.GetUsersByItemID(r.Context(), itemType, itemID)
	if err != nil {
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

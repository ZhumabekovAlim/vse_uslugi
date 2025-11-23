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

	itemResponse, err := h.Service.GetItemResponses(r.Context(), itemType, itemID)
	if err != nil {
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"performer_id": itemResponse.PerformerID,
		"status":       itemResponse.Status,
		"users":        itemResponse.Users,
	}

	switch itemType {
	case "service":
		response["service_id"] = itemResponse.ItemID
		response["service_name"] = itemResponse.ItemName
	case "ad":
		response["ad_id"] = itemResponse.ItemID
		response["ad_name"] = itemResponse.ItemName
	case "rent":
		response["rent_id"] = itemResponse.ItemID
		response["rent_name"] = itemResponse.ItemName
	case "work":
		response["work_id"] = itemResponse.ItemID
		response["work_name"] = itemResponse.ItemName
	case "rent_ad":
		response["rent_ad_id"] = itemResponse.ItemID
		response["rent_ad_name"] = itemResponse.ItemName
	case "work_ad":
		response["work_ad_id"] = itemResponse.ItemID
		response["work_ad_name"] = itemResponse.ItemName
	default:
		http.Error(w, "unknown item type", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(response)
}

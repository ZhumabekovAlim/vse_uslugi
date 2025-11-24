package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

// ResponseUsersHandler handles HTTP requests for retrieving users who responded to items.
type ResponseUsersHandler struct {
	Service     *services.ResponseUsersService
	ChatService *services.ChatService
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

	var chatObject map[string]interface{}
	if userID, ok := r.Context().Value("user_id").(int); ok && h.ChatService != nil {
		if chats, err := h.ChatService.GetChatsByUserID(r.Context(), userID); err == nil {
			if chat, found := findChatByTypeAndID(chats, itemType, itemID); found {
				chatObject = buildChatObject(chat)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{"users": users}
	if chatObject != nil {
		response["chatObject"] = chatObject
	}

	json.NewEncoder(w).Encode(response)
}

func findChatByTypeAndID(chats []models.AdChats, itemType string, itemID int) (models.AdChats, bool) {
	for _, chat := range chats {
		switch itemType {
		case "service":
			if chat.ServiceID != nil && *chat.ServiceID == itemID {
				return chat, true
			}
		case "rent_ad":
			if chat.RentAdID != nil && *chat.RentAdID == itemID {
				return chat, true
			}
		case "work_ad":
			if chat.WorkAdID != nil && *chat.WorkAdID == itemID {
				return chat, true
			}
		case "rent":
			if chat.RentID != nil && *chat.RentID == itemID {
				return chat, true
			}
		case "work":
			if chat.WorkID != nil && *chat.WorkID == itemID {
				return chat, true
			}
		default:
			if chat.AdID != nil && *chat.AdID == itemID {
				return chat, true
			}
		}
	}

	return models.AdChats{}, false
}

func buildChatObject(chat models.AdChats) map[string]interface{} {
	result := map[string]interface{}{
		"ad_name":      chat.AdName,
		"status":       chat.Status,
		"performer_id": chat.PerformerID,
		"users":        chat.Users,
	}

	if chat.AdID != nil {
		result["ad_id"] = *chat.AdID
	}
	if chat.ServiceID != nil {
		result["service_id"] = *chat.ServiceID
	}
	if chat.RentAdID != nil {
		result["rentad_id"] = *chat.RentAdID
	}
	if chat.WorkAdID != nil {
		result["workad_id"] = *chat.WorkAdID
	}
	if chat.RentID != nil {
		result["rent_id"] = *chat.RentID
	}
	if chat.WorkID != nil {
		result["work_id"] = *chat.WorkID
	}

	return result
}

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
	if userID, ok := r.Context().Value("user_id").(int); ok {
		chatObject = getChatObject(r.Context(), h.ChatService, userID, itemType, itemID)
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
		"ad_name": chat.AdName,
		"status":  chat.Status,
		"users":   chat.Users,
	}

	performerID := chat.PerformerID
	if performerID == nil {
		performerID = findPerformerID(chat.Users)
	}
	if performerID != nil {
		result["performer_id"] = *performerID
	}

	var adID *int
	switch {
	case chat.AdID != nil:
		adID = chat.AdID
	case chat.ServiceID != nil:
		adID = chat.ServiceID
	case chat.RentAdID != nil:
		adID = chat.RentAdID
	case chat.WorkAdID != nil:
		adID = chat.WorkAdID
	case chat.RentID != nil:
		adID = chat.RentID
	case chat.WorkID != nil:
		adID = chat.WorkID
	}

	if adID != nil {
		result["ad_id"] = *adID
	}

	return result
}

func findPerformerID(users []models.ChatUser) *int {
	for _, user := range users {
		if user.MyRole == "performer" {
			pid := user.ID
			return &pid
		}
	}

	return nil
}

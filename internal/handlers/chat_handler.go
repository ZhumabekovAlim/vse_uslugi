package handlers

import (
	"encoding/json"
	"errors"
	"naimuBack/internal/models"
	service "naimuBack/internal/services"
	"net/http"
	"strconv"
)

type ChatHandler struct {
	ChatService *service.ChatService
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var chat models.Chat
	err := json.NewDecoder(r.Body).Decode(&chat)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdChat, err := h.ChatService.CreateChat(r.Context(), chat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdChat)
}

func (h *ChatHandler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	chat, err := h.ChatService.GetChatByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrCategoryNotFound) {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve chat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func (h *ChatHandler) GetAllChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.ChatService.GetAllChats(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func (h *ChatHandler) GetChatsByUserID(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(idParam)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	chats, err := h.ChatService.GetChatsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(chats))

	for _, chat := range chats {
		item := map[string]interface{}{
			"ad_name":      chat.AdName,
			"status":       chat.Status,
			"performer_id": chat.PerformerID,
			"users":        chat.Users,
		}

		if chat.AdID != nil {
			item["ad_id"] = *chat.AdID
		}
		if chat.ServiceID != nil {
			item["service_id"] = *chat.ServiceID
		}
		if chat.RentAdID != nil {
			item["rentad_id"] = *chat.RentAdID
		}
		if chat.WorkAdID != nil {
			item["workad_id"] = *chat.WorkAdID
		}
		if chat.RentID != nil {
			item["rent_id"] = *chat.RentID
		}
		if chat.WorkID != nil {
			item["work_id"] = *chat.WorkID
		}

		response = append(response, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"chats": response})
}

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	err = h.ChatService.DeleteChat(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

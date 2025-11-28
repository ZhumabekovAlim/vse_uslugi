package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	service "naimuBack/internal/services"
)

type MessageHandler struct {
	MessageService *service.MessageService
}

func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var message models.Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	senderID, _ := r.Context().Value("user_id").(int)
	role, _ := r.Context().Value("role").(string)
	if senderID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if message.SenderID != 0 && message.SenderID != senderID {
		http.Error(w, "Sender mismatch", http.StatusForbidden)
		return
	}
	message.SenderID = senderID
	if message.ReceiverID == 0 {
		http.Error(w, "receiver_id required", http.StatusBadRequest)
		return
	}

	if role == "business_worker" {
		receiverRole, err := h.MessageService.GetUserRole(r.Context(), message.ReceiverID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if receiverRole == "client" {
			http.Error(w, "business workers cannot message clients", http.StatusForbidden)
			return
		}
	}

	createdMessage, err := h.MessageService.CreateMessage(r.Context(), message)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrWorkerClientCommunication) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdMessage)
}

func (h *MessageHandler) GetMessagesForChat(w http.ResponseWriter, r *http.Request) {
	chatIDParam := r.URL.Query().Get(":chatId")
	chatID, err := strconv.Atoi(chatIDParam)
	if err != nil || chatID <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	requesterID, _ := r.Context().Value("user_id").(int)
	requesterRole, _ := r.Context().Value("role").(string)
	if requesterRole == "business_worker" {
		user1ID, user2ID, err := h.MessageService.GetChatParticipants(r.Context(), chatID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if requesterID != user1ID && requesterID != user2ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		otherID := user1ID
		if otherID == requesterID {
			otherID = user2ID
		}
		role, err := h.MessageService.GetUserRole(r.Context(), otherID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if role == "client" {
			http.Error(w, "business workers cannot access client chats", http.StatusForbidden)
			return
		}
	}

	pageParam := r.URL.Query().Get("page")
	pageSizeParam := r.URL.Query().Get("page_size")

	page, err := strconv.Atoi(pageParam)
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeParam)
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	messages, err := h.MessageService.GetMessagesForChat(r.Context(), chatID, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	messageIDParam := r.URL.Query().Get(":messageId")
	messageID, err := strconv.Atoi(messageIDParam)
	if err != nil || messageID <= 0 {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	err = h.MessageService.DeleteMessage(r.Context(), messageID)
	if err != nil {
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MessageHandler) GetMessagesByUserIDs(w http.ResponseWriter, r *http.Request) {
	// Извлечение параметров запроса
	user1Param := r.URL.Query().Get("user1_id")
	user2Param := r.URL.Query().Get("user2_id")
	pageParam := r.URL.Query().Get("page")
	pageSizeParam := r.URL.Query().Get("page_size")

	// Преобразование и валидация идентификаторов пользователей
	user1ID, err := strconv.Atoi(user1Param)
	if err != nil || user1ID <= 0 {
		http.Error(w, "Invalid user1_id", http.StatusBadRequest)
		return
	}
	user2ID, err := strconv.Atoi(user2Param)
	if err != nil || user2ID <= 0 {
		http.Error(w, "Invalid user2_id", http.StatusBadRequest)
		return
	}

	requesterID, _ := r.Context().Value("user_id").(int)
	requesterRole, _ := r.Context().Value("role").(string)
	if requesterRole == "business_worker" {
		if requesterID != user1ID && requesterID != user2ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		otherID := user1ID
		if otherID == requesterID {
			otherID = user2ID
		}
		role, err := h.MessageService.GetUserRole(r.Context(), otherID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if role == "client" {
			http.Error(w, "business workers cannot access client chats", http.StatusForbidden)
			return
		}
	}

	// Преобразование параметров пагинации
	page, err := strconv.Atoi(pageParam)
	if err != nil || page <= 0 {
		page = 1 // значение по умолчанию
	}
	pageSize, err := strconv.Atoi(pageSizeParam)
	if err != nil || pageSize <= 0 {
		pageSize = 10 // значение по умолчанию
	}

	// Получение сообщений через сервис
	messages, err := h.MessageService.GetMessagesByUserIDs(r.Context(), user1ID, user2ID, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

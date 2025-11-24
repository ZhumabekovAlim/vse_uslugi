package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"naimuBack/internal/services"
)

func getChatObject(ctx context.Context, chatService *services.ChatService, userID int, adType string, adID int) map[string]interface{} {
	if chatService == nil || userID == 0 {
		return nil
	}

	chats, err := chatService.GetChatsByUserID(ctx, userID)
	if err != nil {
		return nil
	}

	if chat, found := findChatByTypeAndID(chats, adType, adID); found {
		return buildChatObject(chat)
	}

	return nil
}

func respondWithChatObject(w http.ResponseWriter, base interface{}, chatObject map[string]interface{}) error {
	if chatObject == nil {
		return json.NewEncoder(w).Encode(base)
	}

	data, err := json.Marshal(base)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}

	response["chatObject"] = chatObject
	return json.NewEncoder(w).Encode(response)
}

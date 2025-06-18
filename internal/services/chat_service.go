package services

import (
	"context"
	"errors"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ChatService struct {
	ChatRepo *repositories.ChatRepository
}

func (s *ChatService) CreateChat(ctx context.Context, chat models.Chat) (models.Chat, error) {
	chatID, err := s.ChatRepo.CreateChat(ctx, chat)
	if err != nil {
		return models.Chat{}, err
	}
	chat.ID = chatID
	return chat, nil
}

func (s *ChatService) GetChatByID(ctx context.Context, id int) (models.Chat, error) {
	chat, err := s.ChatRepo.GetChatByID(ctx, id)
	if err != nil {
		return models.Chat{}, err
	}
	if chat.ID == 0 {
		return models.Chat{}, errors.New("chat not found")
	}
	return chat, nil
}

func (s *ChatService) GetAllChats(ctx context.Context) ([]models.Chat, error) {
	return s.ChatRepo.GetAllChats(ctx)
}

func (s *ChatService) DeleteChat(ctx context.Context, id int) error {
	return s.ChatRepo.DeleteChat(ctx, id)
}

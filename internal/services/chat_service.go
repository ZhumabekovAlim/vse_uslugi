package services

import (
	"context"
	"errors"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ChatService struct {
	ChatRepo     *repositories.ChatRepository
	BusinessRepo *repositories.BusinessRepository
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

func (s *ChatService) GetChatsByUserID(ctx context.Context, userID int) ([]models.AdChats, error) {
	return s.ChatRepo.GetChatsByUserID(ctx, userID)
}

func (s *ChatService) DeleteChat(ctx context.Context, id int) error {
	return s.ChatRepo.DeleteChat(ctx, id)
}

// GetWorkerChats returns base chats between business and its workers.
func (s *ChatService) GetWorkerChats(ctx context.Context, userID int, role string) ([]models.BusinessWorkerChat, error) {
	if s.ChatRepo == nil {
		return nil, errors.New("chat repo not configured")
	}

	switch role {
	case "business":
		return s.ChatRepo.GetBusinessWorkerChats(ctx, userID)
	case models.RoleBusinessWorker:
		if s.BusinessRepo == nil {
			return nil, errors.New("business repo not configured")
		}
		worker, err := s.BusinessRepo.GetWorkerByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if worker.ID == 0 {
			return nil, errors.New("business worker not found")
		}
		chat, err := s.ChatRepo.GetBusinessWorkerChatForWorker(ctx, worker.BusinessUserID, userID)
		if err != nil {
			return nil, err
		}
		if chat == nil {
			return nil, errors.New("chat not found")
		}
		return []models.BusinessWorkerChat{*chat}, nil
	default:
		return nil, errors.New("forbidden")
	}
}

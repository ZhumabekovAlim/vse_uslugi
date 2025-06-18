package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type MessageService struct {
	MessageRepo *repositories.MessageRepository
}

func (s *MessageService) CreateMessage(ctx context.Context, message models.Message) (models.Message, error) {
	messageID, err := s.MessageRepo.CreateMessage(ctx, message)
	if err != nil {
		return models.Message{}, err
	}
	message.ID = messageID
	return message, nil
}

func (s *MessageService) GetMessagesForChat(ctx context.Context, chatID int) ([]models.Message, error) {
	return s.MessageRepo.GetMessagesForChat(ctx, chatID)
}

func (s *MessageService) DeleteMessage(ctx context.Context, messageID int) error {
	return s.MessageRepo.DeleteMessage(ctx, messageID)
}

func (s *MessageService) GetMessagesByUserIDs(ctx context.Context, user1ID, user2ID, page, pageSize int) ([]models.Message, error) {
	messages, err := s.MessageRepo.GetMessagesByUserIDs(ctx, user1ID, user2ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

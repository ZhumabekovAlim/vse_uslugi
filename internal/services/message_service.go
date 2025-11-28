package services

import (
	"context"
	"errors"
	"fmt"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

var ErrWorkerClientCommunication = errors.New("business workers cannot message clients")

type MessageService struct {
	MessageRepo *repositories.MessageRepository
	UserRepo    *repositories.UserRepository
}

func (s *MessageService) CreateMessage(ctx context.Context, message models.Message) (models.Message, error) {
	if err := s.enforceWorkerPolicy(ctx, message.SenderID, message.ReceiverID); err != nil {
		return models.Message{}, err
	}

	messageID, err := s.MessageRepo.CreateMessage(ctx, message)
	if err != nil {
		return models.Message{}, err
	}
	message.ID = messageID
	return message, nil
}

func (s *MessageService) GetMessagesForChat(ctx context.Context, chatID, page, pageSize int) ([]models.Message, error) {
	return s.MessageRepo.GetMessagesForChat(ctx, chatID, page, pageSize)
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

// GetChatParticipants returns chat participant ids.
func (s *MessageService) GetChatParticipants(ctx context.Context, chatID int) (int, int, error) {
	return s.MessageRepo.GetChatParticipants(ctx, chatID)
}

// GetUserRole returns the role for provided user id.
func (s *MessageService) GetUserRole(ctx context.Context, userID int) (string, error) {
	if s.UserRepo == nil {
		return "", fmt.Errorf("user repository not configured")
	}
	return s.UserRepo.GetUserRole(ctx, userID)
}

func (s *MessageService) enforceWorkerPolicy(ctx context.Context, senderID, receiverID int) error {
	if s.UserRepo == nil {
		return nil
	}
	senderRole, err := s.UserRepo.GetUserRole(ctx, senderID)
	if err != nil {
		return err
	}
	if senderRole != "business_worker" {
		return nil
	}
	receiverRole, err := s.UserRepo.GetUserRole(ctx, receiverID)
	if err != nil {
		return err
	}
	if receiverRole == "client" {
		return ErrWorkerClientCommunication
	}
	return nil
}

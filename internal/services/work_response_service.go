package services

import (
	"context"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkResponseService struct {
	WorkResponseRepo *repositories.WorkResponseRepository
	WorkRepo         *repositories.WorkRepository
	ChatRepo         *repositories.ChatRepository
	ConfirmationRepo *repositories.WorkConfirmationRepository
	MessageRepo      *repositories.MessageRepository
}

func (s *WorkResponseService) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	resp, err := s.WorkResponseRepo.CreateWorkResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	work, err := s.WorkRepo.GetWorkByID(ctx, resp.WorkID)
	if err != nil {
		return resp, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: work.UserID, User2ID: resp.UserID})
	if err != nil {
		return resp, err
	}

	resp.ChatID = chatID
	resp.ClientID = work.UserID
	resp.PerformerID = resp.UserID

	_, err = s.ConfirmationRepo.Create(ctx, models.WorkConfirmation{
		WorkID:      resp.WorkID,
		ChatID:      chatID,
		ClientID:    work.UserID,
		PerformerID: resp.UserID,
	})
	if err != nil {
		return resp, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: work.UserID,
		Text:       text,

		ChatID:     chatID,

	}); err != nil {
		return resp, err
	}

	return resp, nil
}

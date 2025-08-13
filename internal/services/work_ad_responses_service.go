package services

import (
	"context"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdResponseService struct {
	WorkAdResponseRepo *repositories.WorkAdResponseRepository
	WorkAdRepo         *repositories.WorkAdRepository
	ChatRepo           *repositories.ChatRepository
	ConfirmationRepo   *repositories.WorkAdConfirmationRepository
	MessageRepo        *repositories.MessageRepository
}

func (s *WorkAdResponseService) CreateWorkAdResponse(ctx context.Context, resp models.WorkAdResponses) (models.WorkAdResponses, error) {
	resp, err := s.WorkAdResponseRepo.CreateWorkAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	work, err := s.WorkAdRepo.GetWorkAdByID(ctx, resp.WorkAdID)
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

	_, err = s.ConfirmationRepo.Create(ctx, models.WorkAdConfirmation{
		WorkAdID:    resp.WorkAdID,
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

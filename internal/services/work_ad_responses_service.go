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
	SubscriptionRepo   *repositories.SubscriptionRepository
}

func (s *WorkAdResponseService) CreateWorkAdResponse(ctx context.Context, resp models.WorkAdResponses) (models.WorkAdResponses, error) {
	resp, err := s.WorkAdResponseRepo.CreateWorkAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.WorkAdResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.WorkAdResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.WorkAdResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	work, err := s.WorkAdRepo.GetWorkAdByID(ctx, resp.WorkAdID, 0)
	if err != nil {
		rollback()
		return models.WorkAdResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: work.UserID, User2ID: resp.UserID})
	if err != nil {
		rollback()
		return models.WorkAdResponses{}, err
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
		rollback()
		return models.WorkAdResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: work.UserID,
		Text:       text,
		ChatID:     chatID,
	}); err != nil {
		rollback()
		return models.WorkAdResponses{}, err
	}

	return resp, nil
}

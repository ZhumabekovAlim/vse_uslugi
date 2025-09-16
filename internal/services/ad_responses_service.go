package services

import (
	"context"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdResponseService struct {
	AdResponseRepo   *repositories.AdResponseRepository
	AdRepo           *repositories.AdRepository
	ChatRepo         *repositories.ChatRepository
	ConfirmationRepo *repositories.AdConfirmationRepository
	MessageRepo      *repositories.MessageRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *AdResponseService) CreateAdResponse(ctx context.Context, resp models.AdResponses) (models.AdResponses, error) {
	resp, err := s.AdResponseRepo.CreateAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.AdResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.AdResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.AdResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	ad, err := s.AdRepo.GetAdByID(ctx, resp.AdID, 0)
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: ad.UserID, User2ID: resp.UserID})
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = ad.UserID
	resp.PerformerID = resp.UserID

	_, err = s.ConfirmationRepo.Create(ctx, models.AdConfirmation{
		AdID:        resp.AdID,
		ChatID:      chatID,
		ClientID:    ad.UserID,
		PerformerID: resp.UserID,
	})
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: ad.UserID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	return resp, nil
}

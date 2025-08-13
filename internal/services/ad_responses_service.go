package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdResponseService struct {
	AdResponseRepo   *repositories.AdResponseRepository
	AdRepo           *repositories.AdRepository
	ChatRepo         *repositories.ChatRepository
	ConfirmationRepo *repositories.AdConfirmationRepository
}

func (s *AdResponseService) CreateAdResponse(ctx context.Context, resp models.AdResponses) (models.AdResponses, error) {
	resp, err := s.AdResponseRepo.CreateAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	ad, err := s.AdRepo.GetAdByID(ctx, resp.AdID)
	if err != nil {
		return resp, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: ad.UserID, User2ID: resp.UserID})
	if err != nil {
		return resp, err
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
		return resp, err
	}

	return resp, nil
}

package services

import (
	"context"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdResponseService struct {
	RentAdResponseRepo *repositories.RentAdResponseRepository
	RentAdRepo         *repositories.RentAdRepository
	ChatRepo           *repositories.ChatRepository
	ConfirmationRepo   *repositories.RentAdConfirmationRepository
	MessageRepo        *repositories.MessageRepository
}

func (s *RentAdResponseService) CreateRentAdResponse(ctx context.Context, resp models.RentAdResponses) (models.RentAdResponses, error) {
	resp, err := s.RentAdResponseRepo.CreateRentAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	rent, err := s.RentAdRepo.GetRentAdByID(ctx, resp.RentAdID)
	if err != nil {
		return resp, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: rent.UserID, User2ID: resp.UserID})
	if err != nil {
		return resp, err
	}

	resp.ChatID = chatID
	resp.ClientID = rent.UserID
	resp.PerformerID = resp.UserID

	_, err = s.ConfirmationRepo.Create(ctx, models.RentAdConfirmation{
		RentAdID:    resp.RentAdID,
		ChatID:      chatID,
		ClientID:    rent.UserID,
		PerformerID: resp.UserID,
	})
	if err != nil {
		return resp, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: rent.UserID,
		Text:       text,
	}); err != nil {
		return resp, err
	}

	return resp, nil
}

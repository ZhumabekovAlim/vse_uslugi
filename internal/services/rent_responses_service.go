package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentResponseService struct {
	RentResponseRepo *repositories.RentResponseRepository
	RentRepo         *repositories.RentRepository
	ChatRepo         *repositories.ChatRepository
	ConfirmationRepo *repositories.RentConfirmationRepository
	MessageRepo      *repositories.MessageRepository
}

func (s *RentResponseService) CreateRentResponse(ctx context.Context, resp models.RentResponses) (models.RentResponses, error) {
	resp, err := s.RentResponseRepo.CreateRentResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	rent, err := s.RentRepo.GetRentByID(ctx, resp.RentID, 0)
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

	_, err = s.ConfirmationRepo.Create(ctx, models.RentConfirmation{
		RentID:      resp.RentID,
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

		ChatID: chatID,
	}); err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *RentResponseService) CancelRentResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.RentResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrNoRecord
		}
		return err
	}
	if resp.UserID != userID {
		return models.ErrForbidden
	}
	if err := s.RentResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.RentID, userID); err != nil {
		return err
	}
	return nil
}

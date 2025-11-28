package services

import (
	"context"
	"database/sql"
	"errors"
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
	SubscriptionRepo   *repositories.SubscriptionRepository
	UserRepo           *repositories.UserRepository
	BusinessRepo       *repositories.BusinessRepository
}

func (s *RentAdResponseService) CreateRentAdResponse(ctx context.Context, resp models.RentAdResponses) (models.RentAdResponses, error) {
	if err := ensureExecutorCanRespond(ctx, s.SubscriptionRepo, resp.UserID, models.SubscriptionTypeService); err != nil {
		return models.RentAdResponses{}, err
	}

	resp, err := s.RentAdResponseRepo.CreateRentAdResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.RentAdResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.RentAdResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.RentAdResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	rent, err := s.RentAdRepo.GetRentAdByID(ctx, resp.RentAdID, 0)
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	clientID, err := resolveBusinessContact(ctx, s.BusinessRepo, rent.UserID)
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	performerID, err := resolveBusinessContact(ctx, s.BusinessRepo, resp.UserID)
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	client, err := s.UserRepo.GetUserByID(ctx, clientID)
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: clientID, User2ID: performerID})
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = clientID
	resp.PerformerID = performerID
	resp.Phone = client.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.RentAdConfirmation{
		RentAdID:    resp.RentAdID,
		ChatID:      chatID,
		ClientID:    clientID,
		PerformerID: performerID,
	})
	if err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   performerID,
		ReceiverID: clientID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.RentAdResponses{}, err
	}

	return resp, nil
}

func (s *RentAdResponseService) CancelRentAdResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.RentAdResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.RentAdResponseRepo.GetByRentAdAndUser(ctx, responseID, userID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return models.ErrNoRecord
				}
				return err
			}
			responseID = resp.ID
		} else {
			return err
		}
	}
	if resp.UserID != userID {
		return models.ErrForbidden
	}
	if err := s.RentAdResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.RentAdID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

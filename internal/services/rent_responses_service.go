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
	SubscriptionRepo *repositories.SubscriptionRepository
	UserRepo         *repositories.UserRepository
	BusinessRepo     *repositories.BusinessRepository
}

func (s *RentResponseService) CreateRentResponse(ctx context.Context, resp models.RentResponses) (models.RentResponses, error) {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, resp.UserID, true)
	if err != nil {
		return models.RentResponses{}, err
	}
	resp.UserID = responderID

	resp, err := s.RentResponseRepo.CreateRentResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.RentResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.RentResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.RentResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	rent, err := s.RentRepo.GetRentByID(ctx, resp.RentID, 0)
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	clientID, err := resolveBusinessContact(ctx, s.BusinessRepo, rent.UserID)
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	performerID, err := resolveBusinessContact(ctx, s.BusinessRepo, resp.UserID)
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	client, err := s.UserRepo.GetUserByID(ctx, clientID)
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: clientID, User2ID: performerID})
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = clientID
	resp.PerformerID = performerID
	resp.Phone = client.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.RentConfirmation{
		RentID:      resp.RentID,
		ChatID:      chatID,
		ClientID:    clientID,
		PerformerID: performerID,
	})
	if err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   performerID,
		ReceiverID: clientID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.RentResponses{}, err
	}

	return resp, nil
}

func (s *RentResponseService) CancelRentResponse(ctx context.Context, responseID, userID int) error {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, userID, false)
	if err != nil {
		return err
	}
	userID = responderID

	resp, err := s.RentResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.RentResponseRepo.GetByRentAndUser(ctx, responseID, userID)
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
	if err := s.RentResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.RentID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

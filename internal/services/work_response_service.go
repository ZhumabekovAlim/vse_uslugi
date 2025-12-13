package services

import (
	"context"
	"database/sql"
	"errors"
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
	SubscriptionRepo *repositories.SubscriptionRepository
	UserRepo         *repositories.UserRepository
	BusinessRepo     *repositories.BusinessRepository
}

func (s *WorkResponseService) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, resp.UserID, true)
	if err != nil {
		return models.WorkResponses{}, err
	}
	resp.UserID = responderID

	resp, err := s.WorkResponseRepo.CreateWorkResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.WorkResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.WorkResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.WorkResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	work, err := s.WorkRepo.GetWorkByID(ctx, resp.WorkID, 0)
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	performerID, err := resolveBusinessContact(ctx, s.BusinessRepo, work.UserID)
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	clientID, err := resolveBusinessContact(ctx, s.BusinessRepo, resp.UserID)
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	performer, err := s.UserRepo.GetUserByID(ctx, performerID)
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: performerID, User2ID: clientID})
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = clientID
	resp.PerformerID = performerID
	resp.Phone = performer.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.WorkConfirmation{
		WorkID:      resp.WorkID,
		ChatID:      chatID,
		ClientID:    clientID,
		PerformerID: performerID,
	})
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   clientID,
		ReceiverID: performerID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	return resp, nil
}

func (s *WorkResponseService) CancelWorkResponse(ctx context.Context, responseID, userID int) error {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, userID, false)
	if err != nil {
		return err
	}
	userID = responderID

	resp, err := s.WorkResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.WorkResponseRepo.GetByWorkAndUser(ctx, responseID, userID)
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
	if err := s.WorkResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.WorkID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

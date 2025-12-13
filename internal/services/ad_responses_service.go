package services

import (
	"context"
	"database/sql"
	"errors"
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
	UserRepo         *repositories.UserRepository
	BusinessRepo     *repositories.BusinessRepository
}

func (s *AdResponseService) CreateAdResponse(ctx context.Context, resp models.AdResponses) (models.AdResponses, error) {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, resp.UserID, true)
	if err != nil {
		return models.AdResponses{}, err
	}
	resp.UserID = responderID

	if err := ensureExecutorCanRespond(ctx, s.SubscriptionRepo, resp.UserID, models.SubscriptionTypeService); err != nil {
		return models.AdResponses{}, err
	}

	resp, err = s.AdResponseRepo.CreateAdResponse(ctx, resp)
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

	clientID, err := resolveBusinessContact(ctx, s.BusinessRepo, ad.UserID)
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	performerID, err := resolveBusinessContact(ctx, s.BusinessRepo, resp.UserID)
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	client, err := s.UserRepo.GetUserByID(ctx, clientID)
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: clientID, User2ID: performerID})
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = clientID
	resp.PerformerID = performerID
	resp.Phone = client.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.AdConfirmation{
		AdID:        resp.AdID,
		ChatID:      chatID,
		ClientID:    clientID,
		PerformerID: performerID,
	})
	if err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   performerID,
		ReceiverID: clientID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.AdResponses{}, err
	}

	return resp, nil
}

func (s *AdResponseService) CancelAdResponse(ctx context.Context, responseID, userID int) error {
	responderID, err := resolveResponderID(ctx, s.BusinessRepo, userID, false)
	if err != nil {
		return err
	}
	userID = responderID

	resp, err := s.AdResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.AdResponseRepo.GetByAdAndUser(ctx, responseID, userID)
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
	if err := s.AdResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.AdID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

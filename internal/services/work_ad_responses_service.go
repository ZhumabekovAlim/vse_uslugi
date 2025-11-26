package services

import (
	"context"
	"database/sql"
	"errors"
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
	UserRepo           *repositories.UserRepository
}

func (s *WorkAdResponseService) CreateWorkAdResponse(ctx context.Context, resp models.WorkAdResponses) (models.WorkAdResponses, error) {
	if err := ensureExecutorCanRespond(ctx, s.SubscriptionRepo, resp.UserID, models.SubscriptionTypeService); err != nil {
		return models.WorkAdResponses{}, err
	}

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

	client, err := s.UserRepo.GetUserByID(ctx, work.UserID)
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
	resp.Phone = client.Phone

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

func (s *WorkAdResponseService) CancelWorkAdResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.WorkAdResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.WorkAdResponseRepo.GetByWorkAdAndUser(ctx, responseID, userID)
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
	if err := s.WorkAdResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.WorkAdID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

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
}

func (s *WorkResponseService) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	if err := ensureExecutorCanRespond(ctx, s.SubscriptionRepo, resp.UserID, models.SubscriptionTypeWork); err != nil {
		return models.WorkResponses{}, err
	}

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

	performer, err := s.UserRepo.GetUserByID(ctx, work.UserID)
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: work.UserID, User2ID: resp.UserID})
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	resp.ChatID = chatID
	resp.ClientID = resp.UserID
	resp.PerformerID = work.UserID
	resp.Phone = performer.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.WorkConfirmation{
		WorkID:      resp.WorkID,
		ChatID:      chatID,
		ClientID:    resp.UserID,
		PerformerID: work.UserID,
	})
	if err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: work.UserID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.WorkResponses{}, err
	}

	return resp, nil
}

func (s *WorkResponseService) CancelWorkResponse(ctx context.Context, responseID, userID int) error {
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

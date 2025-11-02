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
}

func (s *WorkResponseService) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	resp, err := s.WorkResponseRepo.CreateWorkResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	work, err := s.WorkRepo.GetWorkByID(ctx, resp.WorkID, 0)
	if err != nil {
		return resp, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: work.UserID, User2ID: resp.UserID})
	if err != nil {
		return resp, err
	}

	resp.ChatID = chatID
	resp.ClientID = resp.UserID
	resp.PerformerID = work.UserID

	_, err = s.ConfirmationRepo.Create(ctx, models.WorkConfirmation{
		WorkID:      resp.WorkID,
		ChatID:      chatID,
		ClientID:    resp.UserID,
		PerformerID: work.UserID,
	})
	if err != nil {
		return resp, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: work.UserID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *WorkResponseService) CancelWorkResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.WorkResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrNoRecord
		}
		return err
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
	return nil
}

package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ServiceResponseService struct {
	ServiceResponseRepo *repositories.ServiceResponseRepository
	ServiceRepo         *repositories.ServiceRepository
	ChatRepo            *repositories.ChatRepository
	ConfirmationRepo    *repositories.ServiceConfirmationRepository
	MessageRepo         *repositories.MessageRepository
}

func (s *ServiceResponseService) CreateServiceResponse(ctx context.Context, resp models.ServiceResponses) (models.ServiceResponses, error) {

	resp, err := s.ServiceResponseRepo.CreateResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	service, err := s.ServiceRepo.GetServiceByID(ctx, resp.ServiceID, 0)
	if err != nil {
		return resp, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: service.UserID, User2ID: resp.UserID})
	if err != nil {
		return resp, err
	}
	resp.ChatID = chatID
	resp.ClientID = resp.UserID
	resp.PerformerID = service.UserID

	_, err = s.ConfirmationRepo.Create(ctx, models.ServiceConfirmation{
		ServiceID:   resp.ServiceID,
		ChatID:      chatID,
		ClientID:    resp.UserID,
		PerformerID: service.UserID,
	})
	if err != nil {
		return resp, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: service.UserID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		return resp, err
	}
	return resp, nil

}

func (s *ServiceResponseService) CancelServiceResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.ServiceResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrNoRecord
		}
		return err
	}
	if resp.UserID != userID {
		return models.ErrForbidden
	}
	if err := s.ServiceResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.ServiceID, userID); err != nil {
		return err
	}
	return nil
}

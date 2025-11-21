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
	SubscriptionRepo    *repositories.SubscriptionRepository
	UserRepo            *repositories.UserRepository
}

func (s *ServiceResponseService) CreateServiceResponse(ctx context.Context, resp models.ServiceResponses) (models.ServiceResponses, error) {
	if err := ensureExecutorCanRespond(ctx, s.SubscriptionRepo, resp.UserID, models.SubscriptionTypeService); err != nil {
		return models.ServiceResponses{}, err
	}

	resp, err := s.ServiceResponseRepo.CreateResponse(ctx, resp)
	if err != nil {
		return resp, err
	}

	if err := s.SubscriptionRepo.ConsumeResponse(ctx, resp.UserID); err != nil {
		_ = s.ServiceResponseRepo.DeleteResponse(ctx, resp.ID)
		return models.ServiceResponses{}, err
	}

	rollback := func() {
		_ = s.SubscriptionRepo.RestoreResponse(ctx, resp.UserID)
		_ = s.ServiceResponseRepo.DeleteResponse(ctx, resp.ID)
	}

	service, err := s.ServiceRepo.GetServiceByID(ctx, resp.ServiceID, 0)
	if err != nil {
		rollback()
		return models.ServiceResponses{}, err
	}

	performer, err := s.UserRepo.GetUserByID(ctx, service.UserID)
	if err != nil {
		rollback()
		return models.ServiceResponses{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: service.UserID, User2ID: resp.UserID})
	if err != nil {
		rollback()
		return models.ServiceResponses{}, err
	}
	resp.ChatID = chatID
	resp.ClientID = resp.UserID
	resp.PerformerID = service.UserID
	resp.Phone = performer.Phone

	_, err = s.ConfirmationRepo.Create(ctx, models.ServiceConfirmation{
		ServiceID:   resp.ServiceID,
		ChatID:      chatID,
		ClientID:    resp.UserID,
		PerformerID: service.UserID,
	})
	if err != nil {
		rollback()
		return models.ServiceResponses{}, err
	}

	text := fmt.Sprintf("Здравствуйте! Предлагаю цену %v. %s", resp.Price, resp.Description)
	if _, err = s.MessageRepo.CreateMessage(ctx, models.Message{
		SenderID:   resp.UserID,
		ReceiverID: service.UserID,
		Text:       text,

		ChatID: chatID,
	}); err != nil {
		rollback()
		return models.ServiceResponses{}, err
	}
	return resp, nil

}

func (s *ServiceResponseService) CancelServiceResponse(ctx context.Context, responseID, userID int) error {
	resp, err := s.ServiceResponseRepo.GetByID(ctx, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp, err = s.ServiceResponseRepo.GetByServiceAndUser(ctx, responseID, userID)
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
	if err := s.ServiceResponseRepo.DeleteResponse(ctx, responseID); err != nil {
		return err
	}
	if err := s.ConfirmationRepo.DeletePending(ctx, resp.ServiceID, userID); err != nil {
		return err
	}
	if err := s.SubscriptionRepo.RestoreResponse(ctx, userID); err != nil {
		return err
	}
	return nil
}

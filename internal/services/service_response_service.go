package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ServiceResponseService struct {
	ServiceResponseRepo *repositories.ServiceResponseRepository
	ServiceRepo         *repositories.ServiceRepository
	ChatRepo            *repositories.ChatRepository
	ConfirmationRepo    *repositories.ServiceConfirmationRepository
}

func (s *ServiceResponseService) CreateServiceResponse(ctx context.Context, resp models.ServiceResponses) (models.ServiceResponses, error) {

       resp, err := s.ServiceResponseRepo.CreateResponse(ctx, resp)
       if err != nil {
               return resp, err
       }

       service, err := s.ServiceRepo.GetServiceByID(ctx, resp.ServiceID)
       if err != nil {
               return resp, err
       }

       chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: service.UserID, User2ID: resp.UserID})
       if err != nil {
               return resp, err
       }
       resp.ChatID = chatID
       resp.ClientID = service.UserID
       resp.PerformerID = resp.UserID

       _, err = s.ConfirmationRepo.Create(ctx, models.ServiceConfirmation{
               ServiceID:   resp.ServiceID,
               ChatID:      chatID,
               ClientID:    service.UserID,
               PerformerID: resp.UserID,
       })
       if err != nil {
               return resp, err
       }
       return resp, nil

}

package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type ServiceConfirmationService struct {
	ConfirmationRepo *repositories.ServiceConfirmationRepository
}

func (s *ServiceConfirmationService) ConfirmService(ctx context.Context, serviceID, clientID int) error {
	return s.ConfirmationRepo.Confirm(ctx, serviceID, clientID)
}

func (s *ServiceConfirmationService) CancelService(ctx context.Context, serviceID, userID int) error {
	return s.ConfirmationRepo.Cancel(ctx, serviceID, userID)
}

func (s *ServiceConfirmationService) DoneService(ctx context.Context, serviceID int) error {
	return s.ConfirmationRepo.Done(ctx, serviceID)
}

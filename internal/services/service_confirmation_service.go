package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type ServiceConfirmationService struct {
	ConfirmationRepo *repositories.ServiceConfirmationRepository
}

func (s *ServiceConfirmationService) ConfirmService(ctx context.Context, serviceID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, serviceID, performerID)
}

func (s *ServiceConfirmationService) CancelService(ctx context.Context, serviceID int) error {
	return s.ConfirmationRepo.Cancel(ctx, serviceID)
}

func (s *ServiceConfirmationService) DoneService(ctx context.Context, serviceID int) error {
	return s.ConfirmationRepo.Done(ctx, serviceID)
}

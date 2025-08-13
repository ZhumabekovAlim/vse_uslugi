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

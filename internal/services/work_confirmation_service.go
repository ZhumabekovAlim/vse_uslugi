package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type WorkConfirmationService struct {
	ConfirmationRepo *repositories.WorkConfirmationRepository
}

func (s *WorkConfirmationService) ConfirmWork(ctx context.Context, workID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, workID, performerID)
}

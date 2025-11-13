package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type WorkConfirmationService struct {
	ConfirmationRepo *repositories.WorkConfirmationRepository
}

func (s *WorkConfirmationService) ConfirmWork(ctx context.Context, workID, clientID int) error {
	return s.ConfirmationRepo.Confirm(ctx, workID, clientID)
}

func (s *WorkConfirmationService) CancelWork(ctx context.Context, workID, userID int) error {
	return s.ConfirmationRepo.Cancel(ctx, workID, userID)
}

func (s *WorkConfirmationService) DoneWork(ctx context.Context, workID int) error {
	return s.ConfirmationRepo.Done(ctx, workID)
}

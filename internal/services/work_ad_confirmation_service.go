package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type WorkAdConfirmationService struct {
	ConfirmationRepo *repositories.WorkAdConfirmationRepository
}

func (s *WorkAdConfirmationService) ConfirmWorkAd(ctx context.Context, workAdID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, workAdID, performerID)
}

func (s *WorkAdConfirmationService) CancelWorkAd(ctx context.Context, workAdID int) error {
	return s.ConfirmationRepo.Cancel(ctx, workAdID)
}

func (s *WorkAdConfirmationService) DoneWorkAd(ctx context.Context, workAdID int) error {
	return s.ConfirmationRepo.Done(ctx, workAdID)
}

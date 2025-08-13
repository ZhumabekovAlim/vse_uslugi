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

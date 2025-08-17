package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type AdConfirmationService struct {
	ConfirmationRepo *repositories.AdConfirmationRepository
}

func (s *AdConfirmationService) ConfirmAd(ctx context.Context, adID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, adID, performerID)
}

func (s *AdConfirmationService) CancelAd(ctx context.Context, adID int) error {
	return s.ConfirmationRepo.Cancel(ctx, adID)
}

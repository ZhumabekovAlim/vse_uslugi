package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type RentAdConfirmationService struct {
	ConfirmationRepo *repositories.RentAdConfirmationRepository
}

func (s *RentAdConfirmationService) ConfirmRentAd(ctx context.Context, rentAdID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, rentAdID, performerID)
}

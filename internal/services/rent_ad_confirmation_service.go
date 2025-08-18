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

func (s *RentAdConfirmationService) CancelRentAd(ctx context.Context, rentAdID int) error {
	return s.ConfirmationRepo.Cancel(ctx, rentAdID)
}

func (s *RentAdConfirmationService) DoneRentAd(ctx context.Context, rentAdID int) error {
	return s.ConfirmationRepo.Done(ctx, rentAdID)
}

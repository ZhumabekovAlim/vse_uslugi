package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type RentConfirmationService struct {
	ConfirmationRepo *repositories.RentConfirmationRepository
}

func (s *RentConfirmationService) ConfirmRent(ctx context.Context, rentID, ClientID int) error {
	return s.ConfirmationRepo.Confirm(ctx, rentID, ClientID)
}

func (s *RentConfirmationService) CancelRent(ctx context.Context, rentID, userID int) error {
	return s.ConfirmationRepo.Cancel(ctx, rentID, userID)
}

func (s *RentConfirmationService) DoneRent(ctx context.Context, rentID int) error {
	return s.ConfirmationRepo.Done(ctx, rentID)
}

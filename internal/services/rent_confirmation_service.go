package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type RentConfirmationService struct {
	ConfirmationRepo *repositories.RentConfirmationRepository
}

func (s *RentConfirmationService) ConfirmRent(ctx context.Context, rentID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, rentID, performerID)
}

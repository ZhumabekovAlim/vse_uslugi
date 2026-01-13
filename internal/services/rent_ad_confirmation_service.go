package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type RentAdConfirmationService struct {
	ConfirmationRepo *repositories.RentAdConfirmationRepository
	RentAdRepo       *repositories.RentAdRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *RentAdConfirmationService) ConfirmRentAd(ctx context.Context, rentAdID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, rentAdID, performerID)
}

func (s *RentAdConfirmationService) CancelRentAd(ctx context.Context, rentAdID, userID int) error {
	status, err := s.RentAdRepo.GetStatus(ctx, rentAdID)
	if err != nil {
		return err
	}
	if err := s.ConfirmationRepo.Cancel(ctx, rentAdID, userID); err != nil {
		return err
	}
	if status == "active" && s.SubscriptionRepo != nil {
		performerIDs, err := s.ConfirmationRepo.GetPerformerIDs(ctx, rentAdID)
		if err != nil {
			return err
		}
		for _, performerID := range performerIDs {
			if err := s.SubscriptionRepo.RestoreResponse(ctx, performerID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *RentAdConfirmationService) DoneRentAd(ctx context.Context, rentAdID int) error {
	return s.ConfirmationRepo.Done(ctx, rentAdID)
}

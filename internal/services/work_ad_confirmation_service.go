package services

import (
	"context"
	"naimuBack/internal/repositories"
)

type WorkAdConfirmationService struct {
	ConfirmationRepo *repositories.WorkAdConfirmationRepository
	WorkAdRepo       *repositories.WorkAdRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *WorkAdConfirmationService) ConfirmWorkAd(ctx context.Context, workAdID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, workAdID, performerID)
}

func (s *WorkAdConfirmationService) CancelWorkAd(ctx context.Context, workAdID, userID int) error {
	status, err := s.WorkAdRepo.GetStatus(ctx, workAdID)
	if err != nil {
		return err
	}
	if err := s.ConfirmationRepo.Cancel(ctx, workAdID, userID); err != nil {
		return err
	}
	if status == "active" && s.SubscriptionRepo != nil {
		performerIDs, err := s.ConfirmationRepo.GetPerformerIDs(ctx, workAdID)
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

func (s *WorkAdConfirmationService) DoneWorkAd(ctx context.Context, workAdID int) error {
	return s.ConfirmationRepo.Done(ctx, workAdID)
}

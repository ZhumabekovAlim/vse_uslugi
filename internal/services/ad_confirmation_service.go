package services

import (
	"context"
	"fmt"
	"naimuBack/internal/repositories"
)

type AdConfirmationService struct {
	ConfirmationRepo *repositories.AdConfirmationRepository
	AdRepo           *repositories.AdRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *AdConfirmationService) ConfirmAd(ctx context.Context, adID, performerID int) error {
	return s.ConfirmationRepo.Confirm(ctx, adID, performerID)
}

func (s *AdConfirmationService) CancelAd(ctx context.Context, adID, userID int) error {
	fmt.Println(adID, userID)
	status, err := s.AdRepo.GetStatus(ctx, adID)
	fmt.Println("status: ", status)
	if err != nil {
		return err
	}
	if err := s.ConfirmationRepo.Cancel(ctx, adID, userID); err != nil {
		fmt.Println("1 ", err)
		return err
	}
	if status == "active" && s.SubscriptionRepo != nil {
		performerIDs, err := s.ConfirmationRepo.GetPerformerIDs(ctx, adID)
		fmt.Println("performerids: ", performerIDs)
		if err != nil {
			return err
		}
		for _, performerID := range performerIDs {
			if err := s.SubscriptionRepo.RestoreResponse(ctx, performerID); err != nil {
				fmt.Println("2 ", err)
				return err
			}
		}
	}
	return nil
}

func (s *AdConfirmationService) DoneAd(ctx context.Context, adID int) error {
	return s.ConfirmationRepo.Done(ctx, adID)
}

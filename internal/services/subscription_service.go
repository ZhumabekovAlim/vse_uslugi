package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type SubscriptionService struct {
	Repo *repositories.SubscriptionRepository
}

func (s *SubscriptionService) GetProfile(ctx context.Context, userID int) (models.SubscriptionProfile, error) {
	slots, err := s.Repo.GetSlots(ctx, userID)
	if err != nil {
		return models.SubscriptionProfile{}, err
	}
	responses, err := s.Repo.GetResponses(ctx, userID)
	if err != nil {
		return models.SubscriptionProfile{}, err
	}
	activeCount, err := s.Repo.CountActiveExecutorListings(ctx, userID)
	if err != nil {
		return models.SubscriptionProfile{}, err
	}

	profile := models.SubscriptionProfile{
		ExecutorListingSlots:        slots.Slots,
		ActiveExecutorListingsCount: activeCount,
		ResponsePacks:               responses.Packs,
		MonthlyResponsesQuota:       responses.MonthlyQuota,
		RemainingResponses:          responses.Remaining,
		MonthlyAmount:               slots.Slots*1000 + responses.Packs*1000,
	}
	profile.Status.Slots = slots.Status
	profile.Status.Responses = responses.Status
	if !slots.RenewsAt.IsZero() {
		profile.RenewsAt = &slots.RenewsAt
	}
	return profile, nil
}

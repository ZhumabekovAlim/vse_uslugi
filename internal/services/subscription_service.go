package services

import (
	"context"
	"time"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type SubscriptionService struct {
	Repo *repositories.SubscriptionRepository
}

func (s *SubscriptionService) GetProfile(ctx context.Context, userID int) (models.SubscriptionProfile, error) {
	subs, err := s.Repo.ListExecutorSubscriptions(ctx, userID)
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
		Service: buildSubscriptionInfo(subs, models.SubscriptionTypeService),
		Rent:    buildSubscriptionInfo(subs, models.SubscriptionTypeRent),
		Work:    buildSubscriptionInfo(subs, models.SubscriptionTypeWork),
		Responses: models.SubscriptionResponsesSummary{
			Remaining:    responses.Remaining,
			MonthlyQuota: responses.MonthlyQuota,
			Status:       responses.Status,
		},
		ActiveExecutorListingsCount: activeCount,
	}
	if !responses.RenewsAt.IsZero() {
		profile.Responses.RenewsAt = &responses.RenewsAt
	}
	return profile, nil
}

func buildSubscriptionInfo(subs []models.ExecutorSubscription, target models.SubscriptionType) models.SubscriptionInfo {
	info := models.SubscriptionInfo{Type: target}
	now := time.Now()
	for _, sub := range subs {
		if sub.Type != target {
			continue
		}
		info.ExpiresAt = &sub.ExpiresAt
		if sub.ExpiresAt.After(now) {
			info.Active = true
			remaining := int(sub.ExpiresAt.Sub(now).Hours() / 24)
			if remaining < 1 {
				remaining = 1
			}
			info.RemainingDays = remaining
		} else {
			info.Active = false
			info.RemainingDays = 0
		}
		return info
	}
	return info
}

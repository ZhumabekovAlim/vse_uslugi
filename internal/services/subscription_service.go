package services

import (
	"context"
	"fmt"
	"time"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type SubscriptionService struct {
	Repo         *repositories.SubscriptionRepository
	BusinessRepo *repositories.BusinessRepository
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

// ResolveSubscriptionOwnerID determines whose subscription should be fetched based on requester role.
// - business_worker: find the linked business user and return that user ID.
// - business: return the caller's own user ID.
// - everyone else: use the requested user ID unchanged.
func (s *SubscriptionService) ResolveSubscriptionOwnerID(ctx context.Context, requestedUserID int) (int, error) {
	role, _ := ctx.Value("role").(string)
	callerID, _ := ctx.Value("user_id").(int)

	switch role {
	case models.RoleBusinessWorker:
		if s.BusinessRepo == nil {
			return 0, fmt.Errorf("business repository not configured")
		}
		if callerID == 0 {
			return 0, models.ErrForbidden
		}
		worker, err := s.BusinessRepo.GetWorkerByUserID(ctx, callerID)
		if err != nil {
			return 0, err
		}
		if worker.ID == 0 {
			return 0, models.ErrForbidden
		}
		return worker.BusinessUserID, nil
	case "business":
		if callerID == 0 {
			return 0, models.ErrForbidden
		}
		return callerID, nil
	default:
		return requestedUserID, nil
	}
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

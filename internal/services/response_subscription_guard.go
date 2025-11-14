package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

func ensureExecutorCanRespond(ctx context.Context, repo *repositories.SubscriptionRepository, userID int, subType models.SubscriptionType) error {
	if repo == nil {
		return nil
	}

	hasListing, err := repo.HasActiveExecutorListing(ctx, userID, subType)
	if err != nil {
		return err
	}
	if !hasListing {
		return models.ErrNoActiveListings
	}

	responses, err := repo.GetResponses(ctx, userID)
	if err != nil {
		return err
	}
	if responses.Remaining <= 0 {
		return models.ErrNoRemainingResponses
	}
	return nil
}

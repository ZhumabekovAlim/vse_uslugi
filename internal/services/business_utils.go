package services

import (
	"context"
	"database/sql"
	"errors"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// resolveBusinessContact returns the business owner ID for a business worker, otherwise the original userID.
// It allows routing chats and confirmations to the primary business user instead of the worker.
func resolveBusinessContact(ctx context.Context, repo *repositories.BusinessRepository, userID int) (int, error) {
	if repo == nil || userID == 0 {
		return userID, nil
	}

	worker, err := repo.GetWorkerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return userID, nil
		}
		return 0, err
	}
	if worker.ID == 0 {
		return userID, nil
	}
	return worker.BusinessUserID, nil
}

// resolveResponderID returns the business owner ID if the caller is a business worker with respond permission.
// If the caller is not a business worker, the original userID is returned unchanged.
func resolveResponderID(ctx context.Context, repo *repositories.BusinessRepository, userID int, enforcePermission bool) (int, error) {
	if repo == nil || userID == 0 {
		return userID, nil
	}

	role, _ := ctx.Value("role").(string)
	if role != "business_worker" {
		return userID, nil
	}

	worker, err := repo.GetWorkerByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if worker.ID == 0 {
		return 0, models.ErrForbidden
	}
	if enforcePermission && !worker.CanRespond {
		return 0, models.ErrForbidden
	}

	return worker.BusinessUserID, nil
}

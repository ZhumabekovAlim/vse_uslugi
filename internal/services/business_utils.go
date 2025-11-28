package services

import (
	"context"
	"database/sql"
	"errors"

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

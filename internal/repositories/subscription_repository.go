package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type SubscriptionRepository struct {
	DB *sql.DB
}

func (r *SubscriptionRepository) GetSlots(ctx context.Context, userID int) (models.SubscriptionSlots, error) {
	query := `SELECT id, user_id, slots, status, renews_at, provider_subscription_id, created_at, updated_at FROM subscription_slots WHERE user_id = ?`
	var sub models.SubscriptionSlots
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&sub.ID, &sub.UserID, &sub.Slots, &sub.Status, &sub.RenewsAt, &sub.ProviderSubscriptionID, &sub.CreatedAt, &sub.UpdatedAt)
	if err == sql.ErrNoRows {
		return models.SubscriptionSlots{UserID: userID}, nil
	}
	return sub, err
}

func (r *SubscriptionRepository) GetResponses(ctx context.Context, userID int) (models.SubscriptionResponses, error) {
	query := `SELECT id, user_id, packs, status, renews_at, monthly_quota, remaining, provider_subscription_id, created_at, updated_at FROM subscription_responses WHERE user_id = ?`
	var sub models.SubscriptionResponses
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&sub.ID, &sub.UserID, &sub.Packs, &sub.Status, &sub.RenewsAt, &sub.MonthlyQuota, &sub.Remaining, &sub.ProviderSubscriptionID, &sub.CreatedAt, &sub.UpdatedAt)
	if err == sql.ErrNoRows {
		return models.SubscriptionResponses{UserID: userID}, nil
	}
	return sub, err
}

func (r *SubscriptionRepository) CountActiveExecutorListings(ctx context.Context, userID int) (int, error) {
	query := `
        SELECT
            (SELECT COUNT(*) FROM service WHERE user_id = ? AND status = 'active') +
            (SELECT COUNT(*) FROM work_ad WHERE user_id = ? AND status = 'active') +
            (SELECT COUNT(*) FROM rent_ad WHERE user_id = ? AND status = 'active')
    `
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, userID, userID).Scan(&count)
	return count, err
}

func (r *SubscriptionRepository) HasActiveSubscription(ctx context.Context, userID int) (bool, error) {
	query := `SELECT COUNT(*) FROM subscription_slots WHERE user_id = ? AND status = 'active'`
	var count int
	if err := r.DB.QueryRowContext(ctx, query, userID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SubscriptionRepository) ConsumeResponse(ctx context.Context, userID int) error {
	res, err := r.DB.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining - 1 WHERE user_id = ? AND remaining > 0`, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return models.ErrNoRemainingResponses
	}
	return nil
}

func (r *SubscriptionRepository) RestoreResponse(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining + 1 WHERE user_id = ?`, userID)
	return err
}

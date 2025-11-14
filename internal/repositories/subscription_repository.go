package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"naimuBack/internal/models"
)

type SubscriptionRepository struct {
	DB *sql.DB
}

func (r *SubscriptionRepository) ListExecutorSubscriptions(ctx context.Context, userID int) ([]models.ExecutorSubscription, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, user_id, subscription_type, expires_at, created_at, updated_at FROM executor_subscriptions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.ExecutorSubscription
	for rows.Next() {
		sub, err := scanExecutorSubscription(rows)
		if err != nil {
			return nil, err
		}
		if !sub.ExpiresAt.After(time.Now()) {
			if err := r.ArchiveListingsByType(ctx, sub.UserID, sub.Type); err != nil {
				return nil, err
			}
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return subs, nil
}

func scanExecutorSubscription(scanner interface{ Scan(dest ...any) error }) (models.ExecutorSubscription, error) {
	var sub models.ExecutorSubscription
	var subType string
	var updated sql.NullTime
	err := scanner.Scan(&sub.ID, &sub.UserID, &subType, &sub.ExpiresAt, &sub.CreatedAt, &updated)
	if err != nil {
		return models.ExecutorSubscription{}, err
	}
	if updated.Valid {
		t := updated.Time
		sub.UpdatedAt = &t
	}
	sub.Type = models.SubscriptionType(subType)
	return sub, nil
}

func (r *SubscriptionRepository) getExecutorSubscription(ctx context.Context, userID int, subType models.SubscriptionType) (models.ExecutorSubscription, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, user_id, subscription_type, expires_at, created_at, updated_at FROM executor_subscriptions WHERE user_id = ? AND subscription_type = ?`, userID, subType)
	sub, err := scanExecutorSubscription(row)
	if err == sql.ErrNoRows {
		return models.ExecutorSubscription{}, sql.ErrNoRows
	}
	return sub, err
}

func (r *SubscriptionRepository) HasActiveSubscription(ctx context.Context, userID int, subType models.SubscriptionType) (bool, error) {
	sub, err := r.getExecutorSubscription(ctx, userID, subType)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	now := time.Now()
	if !sub.ExpiresAt.After(now) {
		if err := r.ArchiveListingsByType(ctx, userID, subType); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (r *SubscriptionRepository) ArchiveListingsByType(ctx context.Context, userID int, subType models.SubscriptionType) error {
	var query string
	switch subType {
	case models.SubscriptionTypeService:
		query = `UPDATE service SET status = 'archive' WHERE user_id = ? AND status <> 'archive'`
	case models.SubscriptionTypeRent:
		query = `UPDATE rent SET status = 'archive' WHERE user_id = ? AND status <> 'archive'`
	case models.SubscriptionTypeWork:
		query = `UPDATE work SET status = 'archive' WHERE user_id = ? AND status <> 'archive'`
	default:
		return fmt.Errorf("unsupported subscription type: %s", subType)
	}
	_, err := r.DB.ExecContext(ctx, query, userID)
	return err
}

// ArchiveExpiredExecutorListings finds executor subscriptions that have expired by the
// provided moment and archives all related listings for the corresponding type
// (service, rent, work). It returns the number of executor subscription records
// that triggered the archival.
func (r *SubscriptionRepository) ArchiveExpiredExecutorListings(ctx context.Context, now time.Time) (int, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT DISTINCT user_id, subscription_type
FROM executor_subscriptions
WHERE expires_at <= ? AND subscription_type IN ('service', 'rent', 'work')
`, now)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var (
			userID  int
			rawType string
		)
		if err := rows.Scan(&userID, &rawType); err != nil {
			return processed, err
		}
		subType := models.SubscriptionType(rawType)
		if err := r.ArchiveListingsByType(ctx, userID, subType); err != nil {
			return processed, err
		}
		processed++
	}
	if err := rows.Err(); err != nil {
		return processed, err
	}
	return processed, nil
}

func (r *SubscriptionRepository) ExtendSubscription(ctx context.Context, userID int, subType models.SubscriptionType, months int) (models.ExecutorSubscription, error) {
	if months <= 0 {
		return models.ExecutorSubscription{}, fmt.Errorf("months must be positive")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.ExecutorSubscription{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var id int64
	var expiresAt time.Time
	row := tx.QueryRowContext(ctx, `SELECT id, expires_at FROM executor_subscriptions WHERE user_id = ? AND subscription_type = ? FOR UPDATE`, userID, subType)
	switch scanErr := row.Scan(&id, &expiresAt); scanErr {
	case nil:
		base := expiresAt
		now := time.Now()
		if !base.After(now) {
			base = now
		}
		newExpires := base.AddDate(0, months, 0)
		_, err = tx.ExecContext(ctx, `UPDATE executor_subscriptions SET expires_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newExpires, id)
	case sql.ErrNoRows:
		newExpires := time.Now().AddDate(0, months, 0)
		res, execErr := tx.ExecContext(ctx, `INSERT INTO executor_subscriptions (user_id, subscription_type, expires_at) VALUES (?, ?, ?)`, userID, subType, newExpires)
		if execErr != nil {
			err = execErr
			return models.ExecutorSubscription{}, err
		}
		lastID, lastErr := res.LastInsertId()
		if lastErr != nil {
			err = lastErr
			return models.ExecutorSubscription{}, err
		}
		id = lastID
	default:
		err = scanErr
		return models.ExecutorSubscription{}, err
	}

	if err != nil {
		return models.ExecutorSubscription{}, err
	}

	row = tx.QueryRowContext(ctx, `SELECT id, user_id, subscription_type, expires_at, created_at, updated_at FROM executor_subscriptions WHERE id = ?`, id)
	sub, scanErr := scanExecutorSubscription(row)
	if scanErr != nil {
		err = scanErr
		return models.ExecutorSubscription{}, err
	}
	return sub, nil
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
        (SELECT COUNT(*) FROM service WHERE user_id = ? AND status IN ('active', 'pending')) +
        (SELECT COUNT(*) FROM ad WHERE user_id = ? AND status IN ('active', 'pending')) +
        (SELECT COUNT(*) FROM work WHERE user_id = ? AND status IN ('active', 'pending')) +
        (SELECT COUNT(*) FROM work_ad WHERE user_id = ? AND status IN ('active', 'pending')) +
        (SELECT COUNT(*) FROM rent WHERE user_id = ? AND status IN ('active', 'pending')) +
        (SELECT COUNT(*) FROM rent_ad WHERE user_id = ? AND status IN ('active', 'pending'))
`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, userID, userID, userID, userID, userID).Scan(&count)
	return count, err
}

func (r *SubscriptionRepository) HasActiveExecutorListing(ctx context.Context, userID int, subType models.SubscriptionType) (bool, error) {
	var query string
	switch subType {
	case models.SubscriptionTypeService:
		query = `SELECT EXISTS(SELECT 1 FROM service WHERE user_id = ? AND status IN ('active', 'pending'))`
	case models.SubscriptionTypeRent:
		query = `SELECT EXISTS(SELECT 1 FROM rent WHERE user_id = ? AND status IN ('active', 'pending'))`
	case models.SubscriptionTypeWork:
		query = `SELECT EXISTS(SELECT 1 FROM work WHERE user_id = ? AND status IN ('active', 'pending'))`
	default:
		return false, fmt.Errorf("unsupported subscription type: %s", subType)
	}
	var exists bool
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&exists)
	return exists, err
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

func (r *SubscriptionRepository) AddResponsesBalance(ctx context.Context, userID, amount int) (err error) {
	if amount <= 0 {
		return nil
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var id int64
	query := `SELECT id FROM subscription_responses WHERE user_id = ? FOR UPDATE`
	switch scanErr := tx.QueryRowContext(ctx, query, userID).Scan(&id); scanErr {
	case nil:
		_, err = tx.ExecContext(ctx,
			`UPDATE subscription_responses SET remaining = remaining + ?, status = CASE WHEN status = 'inactive' THEN 'active' ELSE status END, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			amount, id,
		)
	case sql.ErrNoRows:
		_, err = tx.ExecContext(ctx,
			`INSERT INTO subscription_responses (user_id, packs, status, renews_at, monthly_quota, remaining) VALUES (?, 0, 'active', ?, 0, ?)`,
			userID, time.Now().UTC(), amount,
		)
	default:
		err = scanErr
	}

	return err
}

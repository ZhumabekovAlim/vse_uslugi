package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type AdConfirmationRepository struct {
	DB *sql.DB
}

func (r *AdConfirmationRepository) Create(ctx context.Context, ac models.AdConfirmation) (models.AdConfirmation, error) {
	query := `INSERT INTO ad_confirmations (ad_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, ac.AdID, ac.ChatID, ac.ClientID, ac.PerformerID, now)
	if err != nil {
		return models.AdConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.AdConfirmation{}, err
	}
	ac.ID = int(id)
	ac.CreatedAt = now
	return ac, nil
}

func (r *AdConfirmationRepository) Confirm(ctx context.Context, adID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE ad_confirmations SET confirmed = true, updated_at = ? WHERE ad_id = ? AND performer_id = ?`, now, adID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM ad_responses WHERE ad_id = ? AND user_id <> ?`, adID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE ad SET status = 'in progress' WHERE id = ?`, adID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AdConfirmationRepository) Cancel(ctx context.Context, adID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE ad SET status = 'active' WHERE id = ?`, adID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ad_confirmations WHERE ad_id = ?`, adID); err != nil {
		return err
	}
	return tx.Commit()
}

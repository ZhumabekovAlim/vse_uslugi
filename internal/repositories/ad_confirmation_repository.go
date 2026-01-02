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
	query := `INSERT INTO ad_confirmations (ad_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
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
	ac.Status = "active"
	ac.CreatedAt = now
	return ac, nil
}

func (r *AdConfirmationRepository) Confirm(ctx context.Context, adID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID int
	query := `SELECT performer_id FROM ad_confirmations WHERE ad_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, adID, performerID, performerID).Scan(&actualPerformerID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE ad_confirmations SET confirmed = true, status = 'in_progress', updated_at = ? WHERE ad_id = ? AND performer_id = ?`, now, adID, actualPerformerID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM ad_responses WHERE ad_id = ? AND user_id <> ?`, adID, actualPerformerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AdConfirmationRepository) Cancel(ctx context.Context, adID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM ad_confirmations WHERE ad_id = ? AND confirmed = true`, adID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE ad_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE ad_id = ? AND client_id = ? AND performer_id = ?`, now, adID, clientID, performerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AdConfirmationRepository) Done(ctx context.Context, adID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE ad_confirmations SET status = 'done', updated_at = ? WHERE ad_id = ? AND confirmed = true`, now, adID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AdConfirmationRepository) DeletePending(ctx context.Context, adID, performerID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM ad_confirmations WHERE ad_id = ? AND performer_id = ? AND confirmed = false`, adID, performerID)
	return err
}

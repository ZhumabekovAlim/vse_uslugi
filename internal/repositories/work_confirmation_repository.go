package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type WorkConfirmationRepository struct {
	DB *sql.DB
}

func (r *WorkConfirmationRepository) Create(ctx context.Context, wc models.WorkConfirmation) (models.WorkConfirmation, error) {
	query := `INSERT INTO work_confirmations (work_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, wc.WorkID, wc.ChatID, wc.ClientID, wc.PerformerID, now)
	if err != nil {
		return models.WorkConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.WorkConfirmation{}, err
	}
	wc.ID = int(id)
	wc.Status = "active"
	wc.CreatedAt = now
	return wc, nil
}

func (r *WorkConfirmationRepository) Confirm(ctx context.Context, workID, clientID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID, actualClientID int
	query := `SELECT performer_id, client_id FROM work_confirmations WHERE work_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, workID, clientID, clientID).Scan(&actualPerformerID, &actualClientID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE work_confirmations SET confirmed = true, status = 'active', updated_at = ? WHERE work_id = ? AND client_id = ?`, now, workID, actualClientID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkConfirmationRepository) Cancel(ctx context.Context, workID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM work_confirmations WHERE work_id = ? AND confirmed = true`, workID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE work_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE work_id = ? AND client_id = ? AND performer_id = ?`, now, workID, clientID, performerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkConfirmationRepository) Done(ctx context.Context, workID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE work_confirmations SET status = 'archived', updated_at = ? WHERE work_id = ? AND confirmed = true`, now, workID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkConfirmationRepository) DeletePending(ctx context.Context, workID, userID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM work_confirmations WHERE work_id = ? AND confirmed = false AND (performer_id = ? OR client_id = ?)`, workID, userID, userID)
	return err
}

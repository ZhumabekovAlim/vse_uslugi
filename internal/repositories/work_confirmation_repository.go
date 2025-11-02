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
	query := `INSERT INTO work_confirmations (work_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
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
	wc.CreatedAt = now
	return wc, nil
}

func (r *WorkConfirmationRepository) Confirm(ctx context.Context, workID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE work_confirmations SET confirmed = true, updated_at = ? WHERE work_id = ? AND performer_id = ?`, now, workID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM work_responses WHERE work_id = ? AND user_id <> ?`, workID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE work SET status = 'in progress' WHERE id = ?`, workID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining - 1 WHERE user_id = ? AND remaining > 0`, performerID)
	if err != nil {
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
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM work_confirmations WHERE work_id = ?`, workID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	if userID == clientID {
		if _, err := tx.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining - 1 WHERE user_id = ? AND remaining > 0`, clientID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining + 1 WHERE user_id = ?`, performerID); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE work SET status = 'active' WHERE id = ?`, workID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM work_confirmations WHERE work_id = ?`, workID); err != nil {
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

	if _, err := tx.ExecContext(ctx, `UPDATE work SET status = 'done' WHERE id = ?`, workID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM work_confirmations WHERE work_id = ?`, workID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkConfirmationRepository) DeletePending(ctx context.Context, workID, performerID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM work_confirmations WHERE work_id = ? AND performer_id = ? AND confirmed = false`, workID, performerID)
	return err
}

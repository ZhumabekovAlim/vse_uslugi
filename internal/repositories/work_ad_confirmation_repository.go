package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type WorkAdConfirmationRepository struct {
	DB *sql.DB
}

func (r *WorkAdConfirmationRepository) Create(ctx context.Context, wc models.WorkAdConfirmation) (models.WorkAdConfirmation, error) {
	query := `INSERT INTO work_ad_confirmations (work_ad_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, wc.WorkAdID, wc.ChatID, wc.ClientID, wc.PerformerID, now)
	if err != nil {
		return models.WorkAdConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.WorkAdConfirmation{}, err
	}
	wc.ID = int(id)
	wc.CreatedAt = now
	return wc, nil
}

func (r *WorkAdConfirmationRepository) Confirm(ctx context.Context, workAdID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE work_ad_confirmations SET confirmed = true, updated_at = ? WHERE work_ad_id = ? AND performer_id = ?`, now, workAdID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM work_ad_responses WHERE work_ad_id = ? AND user_id <> ?`, workAdID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE work_ad SET status = 'active' WHERE id = ?`, workAdID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining - 1 WHERE user_id = ? AND remaining > 0`, performerID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkAdConfirmationRepository) Cancel(ctx context.Context, workAdID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE work_ad SET status = 'active' WHERE id = ?`, workAdID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM work_ad_confirmations WHERE work_ad_id = ?`, workAdID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkAdConfirmationRepository) Done(ctx context.Context, workAdID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE work_ad SET status = 'done' WHERE id = ?`, workAdID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM work_ad_confirmations WHERE work_ad_id = ?`, workAdID); err != nil {
		return err
	}
	return tx.Commit()
}

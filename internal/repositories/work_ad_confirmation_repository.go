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
	query := `INSERT INTO work_ad_confirmations (work_ad_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
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
	wc.Status = "active"
	wc.CreatedAt = now
	return wc, nil
}

func (r *WorkAdConfirmationRepository) Confirm(ctx context.Context, workAdID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID int
	query := `SELECT performer_id FROM work_ad_confirmations WHERE work_ad_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, workAdID, performerID, performerID).Scan(&actualPerformerID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE work_ad_confirmations SET confirmed = true, status = 'in_progress', updated_at = ? WHERE work_ad_id = ? AND performer_id = ?`, now, workAdID, actualPerformerID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE work_ad SET status = 'in_progress', updated_at = ? WHERE id = ?`, now, workAdID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM work_ad_responses WHERE work_ad_id = ? AND user_id <> ?`, workAdID, actualPerformerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkAdConfirmationRepository) Cancel(ctx context.Context, workAdID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM work_ad_confirmations WHERE work_ad_id = ?`, workAdID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE work_ad_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE work_ad_id = ? AND client_id = ? AND performer_id = ?`, now, workAdID, clientID, performerID); err != nil {
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

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE work_ad_confirmations SET status = 'done', updated_at = ? WHERE work_ad_id = ? AND confirmed = true`, now, workAdID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE work_ad SET status = 'done', updated_at = ? WHERE id = ?`, now, workAdID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WorkAdConfirmationRepository) DeletePending(ctx context.Context, workAdID, performerID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM work_ad_confirmations WHERE work_ad_id = ? AND performer_id = ? AND confirmed = false`, workAdID, performerID)
	return err
}

func (r *WorkAdConfirmationRepository) GetPerformerIDs(ctx context.Context, workAdID int) ([]int, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT DISTINCT performer_id FROM work_ad_confirmations WHERE work_ad_id = ?`, workAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var performers []int
	for rows.Next() {
		var performerID int
		if err := rows.Scan(&performerID); err != nil {
			return nil, err
		}
		performers = append(performers, performerID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return performers, nil
}

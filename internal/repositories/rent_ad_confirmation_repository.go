package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type RentAdConfirmationRepository struct {
	DB *sql.DB
}

func (r *RentAdConfirmationRepository) Create(ctx context.Context, rc models.RentAdConfirmation) (models.RentAdConfirmation, error) {
	query := `INSERT INTO rent_ad_confirmations (rent_ad_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, rc.RentAdID, rc.ChatID, rc.ClientID, rc.PerformerID, now)
	if err != nil {
		return models.RentAdConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.RentAdConfirmation{}, err
	}
	rc.ID = int(id)
	rc.Status = "active"
	rc.CreatedAt = now
	return rc, nil
}

func (r *RentAdConfirmationRepository) Confirm(ctx context.Context, rentAdID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID int
	query := `SELECT performer_id FROM rent_ad_confirmations WHERE rent_ad_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, rentAdID, performerID, performerID).Scan(&actualPerformerID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE rent_ad_confirmations SET confirmed = true, status = 'in_progress', updated_at = ? WHERE rent_ad_id = ? AND performer_id = ?`, now, rentAdID, actualPerformerID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rent_ad SET status = 'in_progress', updated_at = ? WHERE id = ?`, now, rentAdID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rent_ad_responses WHERE rent_ad_id = ? AND user_id <> ?`, rentAdID, actualPerformerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentAdConfirmationRepository) Cancel(ctx context.Context, rentAdID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM rent_ad_confirmations WHERE rent_ad_id = ? AND confirmed = true`, rentAdID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE rent_ad_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE rent_ad_id = ? AND client_id = ? AND performer_id = ?`, now, rentAdID, clientID, performerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentAdConfirmationRepository) Done(ctx context.Context, rentAdID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE rent_ad_confirmations SET status = 'done', updated_at = ? WHERE rent_ad_id = ? AND confirmed = true`, now, rentAdID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentAdConfirmationRepository) DeletePending(ctx context.Context, rentAdID, performerID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM rent_ad_confirmations WHERE rent_ad_id = ? AND performer_id = ? AND confirmed = false`, rentAdID, performerID)
	return err
}

func (r *RentAdConfirmationRepository) GetPerformerIDs(ctx context.Context, rentAdID int) ([]int, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT DISTINCT performer_id FROM rent_ad_confirmations WHERE rent_ad_id = ?`, rentAdID)
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

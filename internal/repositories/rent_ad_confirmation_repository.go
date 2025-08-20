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
	query := `INSERT INTO rent_ad_confirmations (rent_ad_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
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
	rc.CreatedAt = now
	return rc, nil
}

func (r *RentAdConfirmationRepository) Confirm(ctx context.Context, rentAdID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE rent_ad_confirmations SET confirmed = true, updated_at = ? WHERE rent_ad_id = ? AND performer_id = ?`, now, rentAdID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM rent_ad_responses WHERE rent_ad_id = ? AND user_id <> ?`, rentAdID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE rent_ad SET status = 'active' WHERE id = ?`, rentAdID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentAdConfirmationRepository) Cancel(ctx context.Context, rentAdID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE rent_ad SET status = 'active' WHERE id = ?`, rentAdID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM rent_ad_confirmations WHERE rent_ad_id = ?`, rentAdID); err != nil {
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

	if _, err := tx.ExecContext(ctx, `UPDATE rent_ad SET status = 'done' WHERE id = ?`, rentAdID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM rent_ad_confirmations WHERE rent_ad_id = ?`, rentAdID); err != nil {
		return err
	}
	return tx.Commit()
}

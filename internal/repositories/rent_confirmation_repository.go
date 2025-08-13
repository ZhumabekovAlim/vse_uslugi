package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type RentConfirmationRepository struct {
	DB *sql.DB
}

func (r *RentConfirmationRepository) Create(ctx context.Context, rc models.RentConfirmation) (models.RentConfirmation, error) {
	query := `INSERT INTO rent_confirmations (rent_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, rc.RentID, rc.ChatID, rc.ClientID, rc.PerformerID, now)
	if err != nil {
		return models.RentConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.RentConfirmation{}, err
	}
	rc.ID = int(id)
	rc.CreatedAt = now
	return rc, nil
}

func (r *RentConfirmationRepository) Confirm(ctx context.Context, rentID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE rent_confirmations SET confirmed = true, updated_at = ? WHERE rent_id = ? AND performer_id = ?`, now, rentID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM rent_responses WHERE rent_id = ? AND user_id <> ?`, rentID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE rent SET status = 'in progress' WHERE id = ?`, rentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

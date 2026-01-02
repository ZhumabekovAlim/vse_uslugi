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
	query := `INSERT INTO rent_confirmations (rent_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
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
	rc.Status = "active"
	rc.CreatedAt = now
	return rc, nil
}

func (r *RentConfirmationRepository) Confirm(ctx context.Context, rentID, ClientID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID, actualClientID int
	query := `SELECT performer_id FROM rent_confirmations WHERE rent_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, rentID, ClientID, ClientID).Scan(&actualPerformerID, &actualClientID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE rent_confirmations SET confirmed = true, status = 'in_progress', updated_at = ? WHERE rent_id = ? AND client_id = ?`, now, rentID, actualClientID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentConfirmationRepository) Cancel(ctx context.Context, rentID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM rent_confirmations WHERE rent_id = ? AND confirmed = true`, rentID).Scan(&clientID, &performerID); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE rent_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE rent_id = ? AND client_id = ? AND performer_id = ?`, now, rentID, clientID, performerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentConfirmationRepository) Done(ctx context.Context, rentID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE rent_confirmations SET status = 'done', updated_at = ? WHERE rent_id = ? AND confirmed = true`, now, rentID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RentConfirmationRepository) DeletePending(ctx context.Context, rentID, performerID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM rent_confirmations WHERE rent_id = ? AND performer_id = ? AND confirmed = false`, rentID, performerID)
	return err
}

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"naimuBack/internal/models"
	"time"
)

var (
	ErrServiceConfirmationNotFound = errors.New("service confirmation not found")
)

type ServiceConfirmationRepository struct {
	DB *sql.DB
}

func (r *ServiceConfirmationRepository) Create(ctx context.Context, sc models.ServiceConfirmation) (models.ServiceConfirmation, error) {
	query := `INSERT INTO service_confirmations (service_id, chat_id, client_id, performer_id, confirmed, created_at) VALUES (?, ?, ?, ?, false, ?)`
	now := time.Now()
	res, err := r.DB.ExecContext(ctx, query, sc.ServiceID, sc.ChatID, sc.ClientID, sc.PerformerID, now)
	if err != nil {
		return models.ServiceConfirmation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.ServiceConfirmation{}, err
	}
	sc.ID = int(id)
	sc.CreatedAt = now
	return sc, nil
}

func (r *ServiceConfirmationRepository) Confirm(ctx context.Context, serviceID, performerID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID, actualClientID int
	query := `SELECT performer_id, client_id FROM service_confirmations WHERE service_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, serviceID, performerID, performerID).Scan(&actualPerformerID, &actualClientID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE service_confirmations SET confirmed = true, updated_at = ? WHERE service_id = ? AND performer_id = ?`, now, serviceID, actualPerformerID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM service_responses WHERE service_id = ? AND user_id <> ?`, serviceID, actualClientID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE service SET status = 'in progress' WHERE id = ?`, serviceID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE subscription_responses SET remaining = remaining - 1 WHERE user_id = ? AND remaining > 0`, actualPerformerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServiceConfirmationRepository) Cancel(ctx context.Context, serviceID, userID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var clientID, performerID int
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM service_confirmations WHERE service_id = ?`, serviceID).Scan(&clientID, &performerID); err != nil {
		if err == sql.ErrNoRows {
			return ErrServiceConfirmationNotFound
		}
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

	if _, err := tx.ExecContext(ctx, `UPDATE service SET status = 'active' WHERE id = ?`, serviceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM service_confirmations WHERE service_id = ?`, serviceID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServiceConfirmationRepository) Done(ctx context.Context, serviceID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE service SET status = 'done' WHERE id = ?`, serviceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM service_confirmations WHERE service_id = ?`, serviceID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServiceConfirmationRepository) DeletePending(ctx context.Context, serviceID, userID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM service_confirmations WHERE service_id = ? AND confirmed = false AND (performer_id = ? OR client_id = ?)`, serviceID, userID, userID)
	return err
}

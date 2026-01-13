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
	query := `INSERT INTO service_confirmations (service_id, chat_id, client_id, performer_id, confirmed, status, created_at) VALUES (?, ?, ?, ?, false, 'active', ?)`
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
	sc.Status = "active"
	sc.CreatedAt = now
	return sc, nil
}

func (r *ServiceConfirmationRepository) Confirm(ctx context.Context, serviceID, clientID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actualPerformerID, actualClientID int
	query := `SELECT performer_id, client_id FROM service_confirmations WHERE service_id = ? AND (performer_id = ? OR client_id = ?)`
	if err := tx.QueryRowContext(ctx, query, serviceID, clientID, clientID).Scan(&actualPerformerID, &actualClientID); err != nil {
		return err
	}

	now := time.Now()
	if _, err = tx.ExecContext(ctx, `UPDATE service_confirmations SET confirmed = true, status = 'in_progress', updated_at = ? WHERE service_id = ? AND client_id = ?`, now, serviceID, actualClientID); err != nil {
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
	if err := tx.QueryRowContext(ctx, `SELECT client_id, performer_id FROM service_confirmations WHERE service_id = ? AND confirmed = true`, serviceID).Scan(&clientID, &performerID); err != nil {
		if err == sql.ErrNoRows {
			return ErrServiceConfirmationNotFound
		}
		return err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE service_confirmations SET confirmed = false, status = 'archived', updated_at = ? WHERE service_id = ? AND client_id = ? AND performer_id = ?`, now, serviceID, clientID, performerID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE service SET status = 'active', updated_at = ? WHERE id = ?`, now, serviceID); err != nil {
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

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `UPDATE service_confirmations SET status = 'done', updated_at = ? WHERE service_id = ? AND confirmed = true`, now, serviceID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServiceConfirmationRepository) DeletePending(ctx context.Context, serviceID, userID int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM service_confirmations WHERE service_id = ? AND confirmed = false AND (performer_id = ? OR client_id = ?)`, serviceID, userID, userID)
	return err
}

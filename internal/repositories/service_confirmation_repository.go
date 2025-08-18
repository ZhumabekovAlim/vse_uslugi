package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
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

	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE service_confirmations SET confirmed = true, updated_at = ? WHERE service_id = ? AND performer_id = ?`, now, serviceID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM service_responses WHERE service_id = ? AND user_id <> ?`, serviceID, performerID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE service SET status = 'in progress' WHERE id = ?`, serviceID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServiceConfirmationRepository) Cancel(ctx context.Context, serviceID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type AdResponseRepository struct {
	DB *sql.DB
}

func (r *AdResponseRepository) CreateAdResponse(ctx context.Context, resp models.AdResponses) (models.AdResponses, error) {
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ad_responses WHERE user_id = ? AND ad_id = ?`, resp.UserID, resp.AdID).Scan(&count); err != nil {
		return models.AdResponses{}, err
	}
	if count > 0 {
		return models.AdResponses{}, models.ErrAlreadyResponded
	}

	query := `
               INSERT INTO ad_responses (user_id, ad_id, price, description, created_at)
               VALUES (?, ?, ?, ?, ?)
       `

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.AdID, resp.Price, resp.Description, now)
	if err != nil {
		return models.AdResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.AdResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

func (r *AdResponseRepository) GetByID(ctx context.Context, id int) (models.AdResponses, error) {
	var resp models.AdResponses
	query := `SELECT id, user_id, ad_id, price, description, created_at, updated_at FROM ad_responses WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&resp.ID,
		&resp.UserID,
		&resp.AdID,
		&resp.Price,
		&resp.Description,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		return models.AdResponses{}, err
	}
	resp.PerformerID = resp.UserID
	return resp, nil
}

func (r *AdResponseRepository) DeleteResponse(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM ad_responses WHERE id = ?`, id)
	return err
}

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

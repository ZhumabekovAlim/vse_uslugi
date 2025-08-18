package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type RentAdResponseRepository struct {
	DB *sql.DB
}

func (r *RentAdResponseRepository) CreateRentAdResponse(ctx context.Context, resp models.RentAdResponses) (models.RentAdResponses, error) {
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM rent_ad_responses WHERE user_id = ? AND rent_ad_id = ?`, resp.UserID, resp.RentAdID).Scan(&count); err != nil {
		return models.RentAdResponses{}, err
	}
	if count > 0 {
		return models.RentAdResponses{}, models.ErrAlreadyResponded
	}

	query := `
               INSERT INTO rent_ad_responses (user_id, rent_ad_id, price, description, created_at)
               VALUES (?, ?, ?, ?, ?)
       `

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.RentAdID, resp.Price, resp.Description, now)
	if err != nil {
		return models.RentAdResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.RentAdResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

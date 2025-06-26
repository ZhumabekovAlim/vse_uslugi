package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type RentResponseRepository struct {
	DB *sql.DB
}

func (r *RentResponseRepository) CreateRentResponse(ctx context.Context, resp models.RentResponses) (models.RentResponses, error) {
	query := `
		INSERT INTO rent_responses (user_id, rent_id, price, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.RentID, resp.Price, resp.Description, now)
	if err != nil {
		return models.RentResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.RentResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

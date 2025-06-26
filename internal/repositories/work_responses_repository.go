package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type WorkResponseRepository struct {
	DB *sql.DB
}

func (r *WorkResponseRepository) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	query := `
		INSERT INTO work_responses (user_id, work_id, price, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.WorkID, resp.Price, resp.Description, now)
	if err != nil {
		return models.WorkResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.WorkResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

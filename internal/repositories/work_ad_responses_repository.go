package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type WorkAdResponseRepository struct {
	DB *sql.DB
}

func (r *WorkAdResponseRepository) CreateWorkAdResponse(ctx context.Context, resp models.WorkAdResponses) (models.WorkAdResponses, error) {
	query := `
		INSERT INTO work_ad_responses (user_id, work_ad_id, price, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.WorkAdID, resp.Price, resp.Description, now)
	if err != nil {
		return models.WorkAdResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.WorkAdResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

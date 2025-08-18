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
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_responses WHERE user_id = ? AND work_id = ?`, resp.UserID, resp.WorkID).Scan(&count); err != nil {
		return models.WorkResponses{}, err
	}
	if count > 0 {
		return models.WorkResponses{}, models.ErrAlreadyResponded
	}

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

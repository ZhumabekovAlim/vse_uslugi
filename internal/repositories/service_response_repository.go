package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type ServiceResponseRepository struct {
	DB *sql.DB
}

func (r *ServiceResponseRepository) CreateResponse(ctx context.Context, resp models.ServiceResponses) (models.ServiceResponses, error) {
	query := `
		INSERT INTO service_responses (user_id, service_id, price, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.DB.ExecContext(ctx, query, resp.UserID, resp.ServiceID, resp.Price, resp.Description, now)
	if err != nil {
		return models.ServiceResponses{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return models.ServiceResponses{}, err
	}

	resp.ID = int(insertedID)
	resp.CreatedAt = now

	return resp, nil
}

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
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM rent_responses WHERE user_id = ? AND rent_id = ?`, resp.UserID, resp.RentID).Scan(&count); err != nil {
		return models.RentResponses{}, err
	}
	if count > 0 {
		return models.RentResponses{}, models.ErrAlreadyResponded
	}

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

func (r *RentResponseRepository) GetByID(ctx context.Context, id int) (models.RentResponses, error) {
	var resp models.RentResponses
	query := `SELECT id, user_id, rent_id, price, description, created_at, updated_at FROM rent_responses WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&resp.ID,
		&resp.UserID,
		&resp.RentID,
		&resp.Price,
		&resp.Description,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		return models.RentResponses{}, err
	}
	resp.PerformerID = resp.UserID
	return resp, nil
}

func (r *RentResponseRepository) GetByRentAndUser(ctx context.Context, rentID, userID int) (models.RentResponses, error) {
	var resp models.RentResponses
	query := `SELECT id, user_id, rent_id, price, description, created_at, updated_at FROM rent_responses WHERE rent_id = ? AND user_id = ?`
	err := r.DB.QueryRowContext(ctx, query, rentID, userID).Scan(
		&resp.ID,
		&resp.UserID,
		&resp.RentID,
		&resp.Price,
		&resp.Description,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		return models.RentResponses{}, err
	}
	resp.PerformerID = resp.UserID
	return resp, nil
}

func (r *RentResponseRepository) DeleteResponse(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM rent_responses WHERE id = ?`, id)
	return err
}

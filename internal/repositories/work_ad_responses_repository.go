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
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_ad_responses WHERE user_id = ? AND work_ad_id = ?`, resp.UserID, resp.WorkAdID).Scan(&count); err != nil {
		return models.WorkAdResponses{}, err
	}
	if count > 0 {
		return models.WorkAdResponses{}, models.ErrAlreadyResponded
	}

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

func (r *WorkAdResponseRepository) GetByID(ctx context.Context, id int) (models.WorkAdResponses, error) {
	var resp models.WorkAdResponses
	query := `SELECT id, user_id, work_ad_id, price, description, created_at, updated_at FROM work_ad_responses WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&resp.ID,
		&resp.UserID,
		&resp.WorkAdID,
		&resp.Price,
		&resp.Description,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		return models.WorkAdResponses{}, err
	}
	resp.PerformerID = resp.UserID
	return resp, nil
}

func (r *WorkAdResponseRepository) GetByWorkAdAndUser(ctx context.Context, workAdID, userID int) (models.WorkAdResponses, error) {
	var resp models.WorkAdResponses
	query := `SELECT id, user_id, work_ad_id, price, description, created_at, updated_at FROM work_ad_responses WHERE work_ad_id = ? AND user_id = ?`
	err := r.DB.QueryRowContext(ctx, query, workAdID, userID).Scan(
		&resp.ID,
		&resp.UserID,
		&resp.WorkAdID,
		&resp.Price,
		&resp.Description,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		return models.WorkAdResponses{}, err
	}
	resp.PerformerID = resp.UserID
	return resp, nil
}

func (r *WorkAdResponseRepository) DeleteResponse(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM work_ad_responses WHERE id = ?`, id)
	return err
}

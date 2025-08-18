package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

type WorkComplaintRepository struct {
	DB *sql.DB
}

func (r *WorkComplaintRepository) CreateWorkComplaint(ctx context.Context, c models.WorkComplaint) error {
	query := `INSERT INTO work_complaints (work_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.WorkID, c.UserID, c.Description)
	return err
}

func (r *WorkComplaintRepository) GetComplaintsByWorkID(ctx context.Context, workID int) ([]models.WorkComplaint, error) {
	query := `SELECT id, work_id, user_id, description, created_at FROM work_complaints WHERE work_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkComplaint
	for rows.Next() {
		var c models.WorkComplaint
		if err := rows.Scan(&c.ID, &c.WorkID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

func (r *WorkComplaintRepository) DeleteWorkComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM work_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *WorkComplaintRepository) GetAllWorkComplaints(ctx context.Context) ([]models.WorkComplaint, error) {
	query := `SELECT id, work_id, user_id, description, created_at FROM work_complaints ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkComplaint
	for rows.Next() {
		var c models.WorkComplaint
		if err := rows.Scan(&c.ID, &c.WorkID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

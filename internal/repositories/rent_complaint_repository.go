package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

type RentComplaintRepository struct {
	DB *sql.DB
}

func (r *RentComplaintRepository) CreateRentComplaint(ctx context.Context, c models.RentComplaint) error {
	query := `INSERT INTO rent_complaints (rent_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.RentID, c.UserID, c.Description)
	return err
}

func (r *RentComplaintRepository) GetComplaintsByRentID(ctx context.Context, rentID int) ([]models.RentComplaint, error) {
	query := `SELECT id, rent_id, user_id, description, created_at FROM rent_complaints WHERE rent_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, rentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentComplaint
	for rows.Next() {
		var c models.RentComplaint
		if err := rows.Scan(&c.ID, &c.RentID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

func (r *RentComplaintRepository) DeleteRentComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM rent_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *RentComplaintRepository) GetAllRentComplaints(ctx context.Context) ([]models.RentComplaint, error) {
	query := `SELECT id, rent_id, user_id, description, created_at FROM rent_complaints ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentComplaint
	for rows.Next() {
		var c models.RentComplaint
		if err := rows.Scan(&c.ID, &c.RentID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

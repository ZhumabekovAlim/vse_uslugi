package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

type RentAdComplaintRepository struct {
	DB *sql.DB
}

func (r *RentAdComplaintRepository) CreateRentAdComplaint(ctx context.Context, c models.RentAdComplaint) error {
	query := `INSERT INTO rent_ad_complaints (rent_ad_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.RentAdID, c.UserID, c.Description)
	return err
}

func (r *RentAdComplaintRepository) GetComplaintsByRentAdID(ctx context.Context, rentAdID int) ([]models.RentAdComplaint, error) {
	query := `SELECT id, rent_ad_id, user_id, description, created_at FROM rent_ad_complaints WHERE rent_ad_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, rentAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentAdComplaint
	for rows.Next() {
		var c models.RentAdComplaint
		if err := rows.Scan(&c.ID, &c.RentAdID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

func (r *RentAdComplaintRepository) DeleteRentAdComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM rent_ad_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *RentAdComplaintRepository) GetAllRentAdComplaints(ctx context.Context) ([]models.RentAdComplaint, error) {
	query := `SELECT id, rent_ad_id, user_id, description, created_at FROM rent_ad_complaints ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentAdComplaint
	for rows.Next() {
		var c models.RentAdComplaint
		if err := rows.Scan(&c.ID, &c.RentAdID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

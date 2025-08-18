package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

type AdComplaintRepository struct {
	DB *sql.DB
}

func (r *AdComplaintRepository) CreateAdComplaint(ctx context.Context, c models.AdComplaint) error {
	query := `INSERT INTO ad_complaints (ad_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.AdID, c.UserID, c.Description)
	return err
}

func (r *AdComplaintRepository) GetComplaintsByAdID(ctx context.Context, adID int) ([]models.AdComplaint, error) {
	query := `SELECT id, ad_id, user_id, description, created_at FROM ad_complaints WHERE ad_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, adID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.AdComplaint
	for rows.Next() {
		var c models.AdComplaint
		if err := rows.Scan(&c.ID, &c.AdID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

func (r *AdComplaintRepository) DeleteAdComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM ad_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *AdComplaintRepository) GetAllAdComplaints(ctx context.Context) ([]models.AdComplaint, error) {
	query := `SELECT id, ad_id, user_id, description, created_at FROM ad_complaints ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.AdComplaint
	for rows.Next() {
		var c models.AdComplaint
		if err := rows.Scan(&c.ID, &c.AdID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

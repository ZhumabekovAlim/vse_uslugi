package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type ComplaintRepository struct {
	DB *sql.DB
}

func (r *ComplaintRepository) CreateComplaint(ctx context.Context, c models.Complaint) error {
	query := `INSERT INTO complaints (service_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.ServiceID, c.UserID, c.Description)
	return err
}

func (r *ComplaintRepository) GetComplaintsByServiceID(ctx context.Context, serviceID int) ([]models.Complaint, error) {
	query := `SELECT id, service_id, user_id, description, created_at FROM complaints WHERE service_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

func (r *ComplaintRepository) DeleteComplaintByID(ctx context.Context, complaintID int) error {
	query := `DELETE FROM complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, complaintID)
	return err
}

func (r *ComplaintRepository) GetAllComplaints(ctx context.Context) ([]models.Complaint, error) {
	query := `SELECT id, service_id, user_id, description, created_at FROM complaints ORDER BY created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.UserID, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

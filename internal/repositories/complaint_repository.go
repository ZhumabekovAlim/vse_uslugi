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
	query := `SELECT c.id, c.service_id, c.user_id, c.description, c.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM complaints c
                JOIN users u ON c.user_id = u.id
                WHERE c.service_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.ServiceID,
			&c.UserID,
			&c.Description,
			&c.CreatedAt,
			&c.User.Name,
			&c.User.Surname,
			&c.User.Email,
			&cityID,
			&avatarPath,
			&reviewRating,
		); err != nil {
			return nil, err
		}
		if cityID.Valid {
			value := int(cityID.Int64)
			c.User.CityID = &value
		}
		if avatarPath.Valid {
			value := avatarPath.String
			c.User.AvatarPath = &value
		}
		if reviewRating.Valid {
			c.User.ReviewRating = reviewRating.Float64
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
	query := `SELECT c.id, c.service_id, c.user_id, c.description, c.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM complaints c
                JOIN users u ON c.user_id = u.id
                ORDER BY c.created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.ServiceID,
			&c.UserID,
			&c.Description,
			&c.CreatedAt,
			&c.User.Name,
			&c.User.Surname,
			&c.User.Email,
			&cityID,
			&avatarPath,
			&reviewRating,
		); err != nil {
			return nil, err
		}
		if cityID.Valid {
			value := int(cityID.Int64)
			c.User.CityID = &value
		}
		if avatarPath.Valid {
			value := avatarPath.String
			c.User.AvatarPath = &value
		}
		if reviewRating.Valid {
			c.User.ReviewRating = reviewRating.Float64
		}
		complaints = append(complaints, c)
	}
	return complaints, nil
}

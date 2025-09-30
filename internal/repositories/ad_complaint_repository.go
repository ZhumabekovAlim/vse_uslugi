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
	query := `SELECT ac.id, ac.ad_id, ac.user_id, ac.description, ac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM ad_complaints ac
                JOIN users u ON ac.user_id = u.id
                WHERE ac.ad_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, adID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.AdComplaint
	for rows.Next() {
		var c models.AdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.AdID,
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

func (r *AdComplaintRepository) DeleteAdComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM ad_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *AdComplaintRepository) GetAllAdComplaints(ctx context.Context) ([]models.AdComplaint, error) {
	query := `SELECT ac.id, ac.ad_id, ac.user_id, ac.description, ac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM ad_complaints ac
                JOIN users u ON ac.user_id = u.id
                ORDER BY ac.created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.AdComplaint
	for rows.Next() {
		var c models.AdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.AdID,
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

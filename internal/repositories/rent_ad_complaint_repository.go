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
	query := `SELECT rac.id, rac.rent_ad_id, rac.user_id, rac.description, rac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM rent_ad_complaints rac
                JOIN users u ON rac.user_id = u.id
                WHERE rac.rent_ad_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, rentAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentAdComplaint
	for rows.Next() {
		var c models.RentAdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.RentAdID,
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

func (r *RentAdComplaintRepository) DeleteRentAdComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM rent_ad_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *RentAdComplaintRepository) GetAllRentAdComplaints(ctx context.Context) ([]models.RentAdComplaint, error) {
	query := `SELECT rac.id, rac.rent_ad_id, rac.user_id, rac.description, rac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM rent_ad_complaints rac
                JOIN users u ON rac.user_id = u.id
                ORDER BY rac.created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentAdComplaint
	for rows.Next() {
		var c models.RentAdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.RentAdID,
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

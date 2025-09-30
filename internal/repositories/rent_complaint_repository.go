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
	query := `SELECT rc.id, rc.rent_id, rc.user_id, rc.description, rc.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM rent_complaints rc
                JOIN users u ON rc.user_id = u.id
                WHERE rc.rent_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, rentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentComplaint
	for rows.Next() {
		var c models.RentComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.RentID,
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

func (r *RentComplaintRepository) DeleteRentComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM rent_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *RentComplaintRepository) GetAllRentComplaints(ctx context.Context) ([]models.RentComplaint, error) {
	query := `SELECT rc.id, rc.rent_id, rc.user_id, rc.description, rc.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM rent_complaints rc
                JOIN users u ON rc.user_id = u.id
                ORDER BY rc.created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.RentComplaint
	for rows.Next() {
		var c models.RentComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.RentID,
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

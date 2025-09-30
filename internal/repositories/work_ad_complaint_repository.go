package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

type WorkAdComplaintRepository struct {
	DB *sql.DB
}

func (r *WorkAdComplaintRepository) CreateWorkAdComplaint(ctx context.Context, c models.WorkAdComplaint) error {
	query := `INSERT INTO work_ad_complaints (work_ad_id, user_id, description) VALUES (?, ?, ?)`
	_, err := r.DB.ExecContext(ctx, query, c.WorkAdID, c.UserID, c.Description)
	return err
}

func (r *WorkAdComplaintRepository) GetComplaintsByWorkAdID(ctx context.Context, workAdID int) ([]models.WorkAdComplaint, error) {
	query := `SELECT wac.id, wac.work_ad_id, wac.user_id, wac.description, wac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM work_ad_complaints wac
                JOIN users u ON wac.user_id = u.id
                WHERE wac.work_ad_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, workAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkAdComplaint
	for rows.Next() {
		var c models.WorkAdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.WorkAdID,
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

func (r *WorkAdComplaintRepository) DeleteWorkAdComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM work_ad_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *WorkAdComplaintRepository) GetAllWorkAdComplaints(ctx context.Context) ([]models.WorkAdComplaint, error) {
	query := `SELECT wac.id, wac.work_ad_id, wac.user_id, wac.description, wac.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM work_ad_complaints wac
                JOIN users u ON wac.user_id = u.id
                ORDER BY wac.created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkAdComplaint
	for rows.Next() {
		var c models.WorkAdComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.WorkAdID,
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

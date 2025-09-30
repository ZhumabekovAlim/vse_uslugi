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
	query := `SELECT wc.id, wc.work_id, wc.user_id, wc.description, wc.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM work_complaints wc
                JOIN users u ON wc.user_id = u.id
                WHERE wc.work_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkComplaint
	for rows.Next() {
		var c models.WorkComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.WorkID,
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

func (r *WorkComplaintRepository) DeleteWorkComplaintByID(ctx context.Context, id int) error {
	query := `DELETE FROM work_complaints WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *WorkComplaintRepository) GetAllWorkComplaints(ctx context.Context) ([]models.WorkComplaint, error) {
	query := `SELECT wc.id, wc.work_id, wc.user_id, wc.description, wc.created_at,
                u.name, u.surname, u.email, u.city_id, u.avatar_path, u.review_rating
                FROM work_complaints wc
                JOIN users u ON wc.user_id = u.id
                ORDER BY wc.created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complaints []models.WorkComplaint
	for rows.Next() {
		var c models.WorkComplaint
		var cityID sql.NullInt64
		var avatarPath sql.NullString
		var reviewRating sql.NullFloat64

		if err := rows.Scan(
			&c.ID,
			&c.WorkID,
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

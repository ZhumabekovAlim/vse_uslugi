package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type WorkReviewRepository struct {
	DB *sql.DB
}

func (r *WorkReviewRepository) CreateWorkReview(ctx context.Context, rev models.WorkReviews) (models.WorkReviews, error) {
	query := `
		INSERT INTO work_reviews (user_id, work_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.WorkID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.WorkReviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.WorkReviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *WorkReviewRepository) GetWorkReviewsByWorkID(ctx context.Context, workID int) ([]models.WorkReviews, error) {
	query := `
               SELECT wr.id, wr.user_id, wr.work_id, wr.rating, wr.review,
                      u.name, u.surname, u.avatar_path,
                      wr.created_at, wr.updated_at
               FROM work_reviews wr
               JOIN users u ON wr.user_id = u.id
               WHERE wr.work_id = ?
               ORDER BY wr.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.WorkReviews{}
	for rows.Next() {
		var rev models.WorkReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.WorkID, &rev.Rating, &rev.Review,
			&rev.UserName, &rev.UserSurname, &rev.UserAvatarPath,
			&rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if rev.UserAvatarPath != nil && *rev.UserAvatarPath == "" {
			rev.UserAvatarPath = nil
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *WorkReviewRepository) UpdateWorkReview(ctx context.Context, rev models.WorkReviews) error {
	query := `
		UPDATE work_reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *WorkReviewRepository) DeleteWorkReview(ctx context.Context, id int) error {
	query := `DELETE FROM work_reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

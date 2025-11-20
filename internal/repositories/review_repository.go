package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type ReviewRepository struct {
	DB *sql.DB
}

func (r *ReviewRepository) CreateReview(ctx context.Context, rev models.Reviews) (models.Reviews, error) {
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews WHERE user_id = ? AND service_id = ?`, rev.UserID, rev.ServiceID).Scan(&count); err != nil {
		return models.Reviews{}, err
	}
	if count > 0 {
		return models.Reviews{}, models.ErrAlreadyReviewed
	}

	query := `
INSERT INTO reviews (user_id, service_id, rating, review, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.ServiceID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.Reviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Reviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *ReviewRepository) GetReviewsByServiceID(ctx context.Context, serviceID int) ([]models.Reviews, error) {
	query := `
               SELECT r.id, r.user_id, r.service_id, r.rating, r.review,
                      u.name, u.surname, u.avatar_path,
                      r.created_at, r.updated_at
               FROM reviews r
               JOIN users u ON r.user_id = u.id
               WHERE r.service_id = ?
               ORDER BY r.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.Reviews{}
	for rows.Next() {
		var rev models.Reviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.ServiceID, &rev.Rating, &rev.Review,
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

func (r *ReviewRepository) UpdateReview(ctx context.Context, rev models.Reviews) error {
	query := `
		UPDATE reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *ReviewRepository) DeleteReview(ctx context.Context, id int) error {
	query := `DELETE FROM reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

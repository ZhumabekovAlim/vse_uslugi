package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type WorkAdReviewRepository struct {
	DB *sql.DB
}

func (r *WorkAdReviewRepository) CreateWorkAdReview(ctx context.Context, rev models.WorkAdReviews) (models.WorkAdReviews, error) {
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_ad_reviews WHERE user_id = ? AND work_ad_id = ?`, rev.UserID, rev.WorkAdID).Scan(&count); err != nil {
		return models.WorkAdReviews{}, err
	}
	if count > 0 {
		return models.WorkAdReviews{}, models.ErrAlreadyReviewed
	}

	query := `
INSERT INTO work_ad_reviews (user_id, work_ad_id, rating, review, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.WorkAdID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.WorkAdReviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.WorkAdReviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *WorkAdReviewRepository) GetWorkAdReviewsByWorkID(ctx context.Context, workAdID int) ([]models.WorkAdReviews, error) {
	query := `
               SELECT war.id, war.user_id, war.work_ad_id, war.rating, war.review,
                      u.name, u.surname, u.avatar_path,
                      war.created_at, war.updated_at
               FROM work_ad_reviews war
               JOIN users u ON war.user_id = u.id
               WHERE war.work_ad_id = ?
               ORDER BY war.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, workAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.WorkAdReviews{}
	for rows.Next() {
		var rev models.WorkAdReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.WorkAdID, &rev.Rating, &rev.Review,
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

func (r *WorkAdReviewRepository) UpdateWorkAdReview(ctx context.Context, rev models.WorkAdReviews) error {
	query := `
		UPDATE work_ad_reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *WorkAdReviewRepository) DeleteWorkAdReview(ctx context.Context, id int) error {
	query := `DELETE FROM work_ad_reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

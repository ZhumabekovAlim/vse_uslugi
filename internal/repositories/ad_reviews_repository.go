package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type AdReviewRepository struct {
	DB *sql.DB
}

func (r *AdReviewRepository) CreateAdReview(ctx context.Context, rev models.AdReviews) (models.AdReviews, error) {
	query := `
		INSERT INTO ad_reviews (user_id, ad_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.AdID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.AdReviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.AdReviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *AdReviewRepository) GetReviewsByAdID(ctx context.Context, adID int) ([]models.AdReviews, error) {
	query := `
		SELECT id, user_id, ad_id, rating, review, created_at, updated_at
		FROM ad_reviews
		WHERE ad_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, adID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.AdReviews{}
	for rows.Next() {
		var rev models.AdReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.AdID, &rev.Rating, &rev.Review, &rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *AdReviewRepository) UpdateAdReview(ctx context.Context, rev models.AdReviews) error {
	query := `
		UPDATE ad_reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *AdReviewRepository) DeleteAdReview(ctx context.Context, id int) error {
	query := `DELETE FROM ad_reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

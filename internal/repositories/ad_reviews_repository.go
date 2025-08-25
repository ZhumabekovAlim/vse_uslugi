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
               SELECT ar.id, ar.user_id, ar.ad_id, ar.rating, ar.review,
                      u.name, u.surname, u.avatar_path,
                      ar.created_at, ar.updated_at
               FROM ad_reviews ar
               JOIN users u ON ar.user_id = u.id
               WHERE ar.ad_id = ?
               ORDER BY ar.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, adID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.AdReviews{}
	for rows.Next() {
		var rev models.AdReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.AdID, &rev.Rating, &rev.Review,
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

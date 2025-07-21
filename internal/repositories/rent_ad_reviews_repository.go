package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type RentAdReviewRepository struct {
	DB *sql.DB
}

func (r *RentAdReviewRepository) CreateRentAdReview(ctx context.Context, rev models.RentAdReviews) (models.RentAdReviews, error) {
	query := `
		INSERT INTO rent_ad_reviews (user_id, rent_ad_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.RentAdID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.RentAdReviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RentAdReviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *RentAdReviewRepository) GetRentAdReviewsByRentID(ctx context.Context, rentAdID int) ([]models.RentAdReviews, error) {
	query := `
		SELECT id, user_id, rent_ad_id, rating, review, created_at, updated_at
		FROM rent_ad_reviews
		WHERE rent_ad_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, rentAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.RentAdReviews{}
	for rows.Next() {
		var rev models.RentAdReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.RentAdID, &rev.Rating, &rev.Review, &rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *RentAdReviewRepository) UpdateRentAdReview(ctx context.Context, rev models.RentAdReviews) error {
	query := `
		UPDATE rent_ad_reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *RentAdReviewRepository) DeleteRentAdReview(ctx context.Context, id int) error {
	query := `DELETE FROM rent_ad_reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

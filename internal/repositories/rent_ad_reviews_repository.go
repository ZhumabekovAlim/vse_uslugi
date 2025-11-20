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
	var count int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM rent_ad_reviews WHERE user_id = ? AND rent_ad_id = ?`, rev.UserID, rev.RentAdID).Scan(&count); err != nil {
		return models.RentAdReviews{}, err
	}
	if count > 0 {
		return models.RentAdReviews{}, models.ErrAlreadyReviewed
	}

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
               SELECT rar.id, rar.user_id, rar.rent_ad_id, rar.rating, rar.review,
                      u.name, u.surname, u.avatar_path,
                      rar.created_at, rar.updated_at
               FROM rent_ad_reviews rar
               JOIN users u ON rar.user_id = u.id
               WHERE rar.rent_ad_id = ?
               ORDER BY rar.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, rentAdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.RentAdReviews{}
	for rows.Next() {
		var rev models.RentAdReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.RentAdID, &rev.Rating, &rev.Review,
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

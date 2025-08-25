package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type RentReviewRepository struct {
	DB *sql.DB
}

func (r *RentReviewRepository) CreateRentReview(ctx context.Context, rev models.RentReviews) (models.RentReviews, error) {
	query := `
		INSERT INTO rent_reviews (user_id, rent_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		rev.UserID, rev.RentID, rev.Rating, rev.Review,
	)
	if err != nil {
		return models.RentReviews{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RentReviews{}, err
	}
	rev.ID = int(id)
	return rev, nil
}

func (r *RentReviewRepository) GetRentReviewsByRentID(ctx context.Context, rentID int) ([]models.RentReviews, error) {
	query := `
               SELECT rr.id, rr.user_id, rr.rent_id, rr.rating, rr.review,
                      u.name, u.surname, u.avatar_path,
                      rr.created_at, rr.updated_at
               FROM rent_reviews rr
               JOIN users u ON rr.user_id = u.id
               WHERE rr.rent_id = ?
               ORDER BY rr.created_at DESC
       `
	rows, err := r.DB.QueryContext(ctx, query, rentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []models.RentReviews{}
	for rows.Next() {
		var rev models.RentReviews
		err := rows.Scan(&rev.ID, &rev.UserID, &rev.RentID, &rev.Rating, &rev.Review,
			&rev.UserName, &rev.UserSurname, &rev.UserAvatarPath,
			&rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *RentReviewRepository) UpdateRentReview(ctx context.Context, rev models.RentReviews) error {
	query := `
		UPDATE rent_reviews
		SET rating = ?, review = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.DB.ExecContext(ctx, query, rev.Rating, rev.Review, rev.ID)
	return err
}

func (r *RentReviewRepository) DeleteRentReview(ctx context.Context, id int) error {
	query := `DELETE FROM rent_reviews WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

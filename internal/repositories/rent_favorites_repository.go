package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"naimuBack/internal/models"
)

type RentFavoriteRepository struct {
	DB *sql.DB
}

func (r *RentFavoriteRepository) AddRentToFavorites(ctx context.Context, fav models.RentFavorite) error {
	query := `INSERT INTO rent_favorites (user_id, rent_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.RentID)
	return err
}

func (r *RentFavoriteRepository) RemoveRentFromFavorites(ctx context.Context, userID, rentID int) error {
	query := `DELETE FROM rent_favorites WHERE user_id = ? AND rent_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, rentID)
	return err
}

func (r *RentFavoriteRepository) IsRentFavorite(ctx context.Context, userID, rentID int) (bool, error) {
	query := `SELECT COUNT(*) FROM rent_favorites WHERE user_id = ? AND rent_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, rentID).Scan(&count)
	return count > 0, err
}

func (r *RentFavoriteRepository) GetRentFavoritesByUser(ctx context.Context, userID int) ([]models.RentFavorite, error) {
	query := `SELECT rf.id, rf.user_id, rf.rent_id, r.city_id, city.name, r.name, r.address, r.price, r.price_to, r.work_time_from, r.work_time_to, r.negotiable, r.hide_phone,
                     u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                     r.status, r.created_at, r.images
                 FROM rent_favorites rf
                 JOIN rent r ON rf.rent_id = r.id
                 JOIN users u ON r.user_id = u.id
                 LEFT JOIN cities city ON r.city_id = city.id
                 WHERE rf.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.RentFavorite
	for rows.Next() {
		var fav models.RentFavorite
		var price, priceTo sql.NullFloat64
		var imagesJSON sql.NullString
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.RentID, &fav.CityID, &fav.CityName, &fav.Name, &fav.Address, &price, &priceTo, &fav.WorkTimeFrom, &fav.WorkTimeTo, &fav.Negotiable, &fav.HidePhone,
			&fav.User.ID, &fav.User.Name, &fav.User.Surname, &fav.User.Phone, &fav.User.ReviewRating, &fav.User.AvatarPath,
			&fav.Status, &fav.CreatedAt, &imagesJSON)
		if err != nil {
			return nil, err
		}
		if price.Valid {
			fav.Price = &price.Float64
		}
		if priceTo.Valid {
			fav.PriceTo = &priceTo.Float64
		}

		if count, err := getUserTotalReviews(ctx, r.DB, fav.User.ID); err == nil {
			fav.User.ReviewsCount = count
		}

		imgPath, err := extractFirstImagePath(imagesJSON)
		if err != nil {
			log.Printf("failed to decode rent images for favorite %d: %v", fav.ID, err)
		}
		fav.ImagePath = imgPath
		favs = append(favs, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rent favorites rows error: %w", err)
	}
	return favs, nil
}

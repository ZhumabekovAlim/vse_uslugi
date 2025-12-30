package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"naimuBack/internal/models"
)

type RentAdFavoriteRepository struct {
	DB *sql.DB
}

func (r *RentAdFavoriteRepository) AddRentAdToFavorites(ctx context.Context, fav models.RentAdFavorite) error {
	query := `INSERT INTO rent_ad_favorites (user_id, rent_ad_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.RentAdID)
	return err
}

func (r *RentAdFavoriteRepository) RemoveRentAdFromFavorites(ctx context.Context, userID, rentAdID int) error {
	query := `DELETE FROM rent_ad_favorites WHERE user_id = ? AND rent_ad_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, rentAdID)
	return err
}

func (r *RentAdFavoriteRepository) IsRentAdFavorite(ctx context.Context, userID, rentAdID int) (bool, error) {
	query := `SELECT COUNT(*) FROM rent_ad_favorites WHERE user_id = ? AND rent_ad_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, rentAdID).Scan(&count)
	return count > 0, err
}

func (r *RentAdFavoriteRepository) GetRentAdFavoritesByUser(ctx context.Context, userID int) ([]models.RentAdFavorite, error) {
	query := `SELECT rf.id, rf.user_id, rf.rent_ad_id, ra.name, ra.price, ra.price_to, ra.work_time_from, ra.work_time_to, ra.negotiable, ra.hide_phone, ra.order_date, ra.order_time, ra.status, ra.created_at, ra.images
                 FROM rent_ad_favorites rf
                 JOIN rent_ad ra ON rf.rent_ad_id = ra.id
                 WHERE rf.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.RentAdFavorite
	for rows.Next() {
		var fav models.RentAdFavorite
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var imagesJSON sql.NullString
		if err := rows.Scan(&fav.ID, &fav.UserID, &fav.RentAdID, &fav.Name, &price, &priceTo, &fav.WorkTimeFrom, &fav.WorkTimeTo, &fav.Negotiable, &fav.HidePhone, &orderDate, &orderTime, &fav.Status, &fav.CreatedAt, &imagesJSON); err != nil {
			return nil, err
		}
		if price.Valid {
			fav.Price = &price.Float64
		}
		if priceTo.Valid {
			fav.PriceTo = &priceTo.Float64
		}
		if orderDate.Valid {
			fav.OrderDate = &orderDate.String
		}
		if orderTime.Valid {
			fav.OrderTime = &orderTime.String
		}

		imgPath, err := extractFirstImagePath(imagesJSON)
		if err != nil {
			log.Printf("failed to decode rent ad images for favorite %d: %v", fav.ID, err)
		}
		fav.ImagePath = imgPath
		favs = append(favs, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rent ad favorites rows error: %w", err)
	}
	return favs, nil
}

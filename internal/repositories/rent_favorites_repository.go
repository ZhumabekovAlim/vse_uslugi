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
	query := `SELECT rf.id, rf.user_id, rf.rent_id, r.name, r.price, r.price_to, r.work_time_from, r.work_time_to, r.negotiable, r.hide_phone, r.status, r.created_at, r.images
                 FROM rent_favorites rf
                 JOIN rent r ON rf.rent_id = r.id
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
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.RentID, &fav.Name, &price, &priceTo, &fav.WorkTimeFrom, &fav.WorkTimeTo, &fav.Negotiable, &fav.HidePhone, &fav.Status, &fav.CreatedAt, &imagesJSON)
		if err != nil {
			return nil, err
		}
		if price.Valid {
			fav.Price = &price.Float64
		}
		if priceTo.Valid {
			fav.PriceTo = &priceTo.Float64
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

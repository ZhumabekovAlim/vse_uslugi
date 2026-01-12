package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"naimuBack/internal/models"
)

type AdFavoriteRepository struct {
	DB *sql.DB
}

func (r *AdFavoriteRepository) AddAdToFavorites(ctx context.Context, fav models.AdFavorite) error {
	query := `INSERT INTO ad_favorites (user_id, ad_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.AdID)
	return err
}

func (r *AdFavoriteRepository) RemoveAdFromFavorites(ctx context.Context, userID, adID int) error {
	query := `DELETE FROM ad_favorites WHERE user_id = ? AND ad_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, adID)
	return err
}

func (r *AdFavoriteRepository) IsAdFavorite(ctx context.Context, userID, adID int) (bool, error) {
	query := `SELECT COUNT(*) FROM ad_favorites WHERE user_id = ? AND ad_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, adID).Scan(&count)
	return count > 0, err
}

func (r *AdFavoriteRepository) GetAdFavoritesByUser(ctx context.Context, userID int) ([]models.AdFavorite, error) {
	query := `SELECT af.id, af.user_id, af.ad_id, a.city_id, a.name, a.price, a.price_to, a.on_site, a.negotiable, a.hide_phone, a.order_date, a.order_time, a.status, a.created_at, a.images
                 FROM ad_favorites af
                 JOIN ad a ON af.ad_id = a.id
                 WHERE af.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.AdFavorite
	for rows.Next() {
		var fav models.AdFavorite
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var imagesJSON sql.NullString
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.AdID, &fav.CityID, &fav.Name, &price, &priceTo, &fav.OnSite, &fav.Negotiable, &fav.HidePhone, &orderDate, &orderTime, &fav.Status, &fav.CreatedAt, &imagesJSON)
		if err != nil {
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
			log.Printf("failed to decode ad images for favorite %d: %v", fav.ID, err)
		}
		fav.ImagePath = imgPath
		favs = append(favs, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ad favorites rows error: %w", err)
	}
	return favs, nil
}

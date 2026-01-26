package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"naimuBack/internal/models"
)

type ServiceFavoriteRepository struct {
	DB *sql.DB
}

func (r *ServiceFavoriteRepository) AddToFavorites(ctx context.Context, fav models.ServiceFavorite) error {
	query := `INSERT INTO service_favorites (user_id, service_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.ServiceID)
	return err
}

func (r *ServiceFavoriteRepository) RemoveFromFavorites(ctx context.Context, userID, serviceID int) error {
	query := `DELETE FROM service_favorites WHERE user_id = ? AND service_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, serviceID)
	return err
}

func (r *ServiceFavoriteRepository) IsFavorite(ctx context.Context, userID, serviceID int) (bool, error) {
	query := `SELECT COUNT(*) FROM service_favorites WHERE user_id = ? AND service_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, serviceID).Scan(&count)
	return count > 0, err
}

func (r *ServiceFavoriteRepository) GetFavoritesByUser(ctx context.Context, userID int) ([]models.ServiceFavorite, error) {
	query := `SELECT sf.id, sf.user_id, sf.service_id, s.city_id, city.name, s.name, s.address, s.price, s.price_to, s.on_site, s.negotiable, s.hide_phone,
                     u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                     s.status, s.created_at, s.images
             FROM service_favorites sf
             JOIN service s ON sf.service_id = s.id
             JOIN users u ON s.user_id = u.id
             LEFT JOIN cities city ON s.city_id = city.id
             WHERE sf.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.ServiceFavorite
	for rows.Next() {
		var fav models.ServiceFavorite
		var price, priceTo sql.NullFloat64
		var imagesJSON sql.NullString
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.ServiceID, &fav.CityID, &fav.CityName, &fav.Name, &fav.Address, &price, &priceTo, &fav.OnSite, &fav.Negotiable, &fav.HidePhone,
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
			log.Printf("failed to decode service images for favorite %d: %v", fav.ID, err)
		}
		fav.ImagePath = imgPath
		favs = append(favs, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("service favorites rows error: %w", err)
	}
	return favs, nil
}

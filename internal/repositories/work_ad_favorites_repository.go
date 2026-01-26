package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"naimuBack/internal/models"
)

type WorkAdFavoriteRepository struct {
	DB *sql.DB
}

func (r *WorkAdFavoriteRepository) AddWorkAdToFavorites(ctx context.Context, fav models.WorkAdFavorite) error {
	query := `INSERT INTO work_ad_favorites (user_id, work_ad_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.WorkAdID)
	return err
}

func (r *WorkAdFavoriteRepository) RemoveWorkAdFromFavorites(ctx context.Context, userID, workAdID int) error {
	query := `DELETE FROM work_ad_favorites WHERE user_id = ? AND work_ad_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, workAdID)
	return err
}

func (r *WorkAdFavoriteRepository) IsWorkAdFavorite(ctx context.Context, userID, workAdID int) (bool, error) {
	query := `SELECT COUNT(*) FROM work_ad_favorites WHERE user_id = ? AND work_ad_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, workAdID).Scan(&count)
	return count > 0, err
}

func (r *WorkAdFavoriteRepository) GetWorkAdFavoritesByUser(ctx context.Context, userID int) ([]models.WorkAdFavorite, error) {
	query := `SELECT wf.id, wf.user_id, wf.work_ad_id, w.city_id, city.name, w.name, w.address, w.price, w.price_to, w.negotiable, w.hide_phone,
                     u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                     w.status, w.created_at, w.images
                 FROM work_ad_favorites wf
                 JOIN work_ad w ON wf.work_ad_id = w.id
                 JOIN users u ON w.user_id = u.id
                 LEFT JOIN cities city ON w.city_id = city.id
                 WHERE wf.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.WorkAdFavorite
	for rows.Next() {
		var fav models.WorkAdFavorite
		var price, priceTo sql.NullFloat64
		var imagesJSON sql.NullString
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.WorkAdID, &fav.CityID, &fav.CityName, &fav.Name, &fav.Address, &price, &priceTo, &fav.Negotiable, &fav.HidePhone,
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
			log.Printf("failed to decode work ad images for favorite %d: %v", fav.ID, err)
		}
		fav.ImagePath = imgPath
		favs = append(favs, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("work ad favorites rows error: %w", err)
	}
	return favs, nil
}

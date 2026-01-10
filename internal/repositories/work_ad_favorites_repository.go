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

func (r *WorkAdFavoriteRepository) GetWorkAdFavoritesByUser(ctx context.Context, userID int, cityID int) ([]models.WorkAdFavorite, error) {
	query := `SELECT wf.id, wf.user_id, wf.work_ad_id, w.name, w.price, w.price_to, w.negotiable, w.hide_phone, w.status, w.created_at, w.images
                 FROM work_ad_favorites wf
                 JOIN work_ad w ON wf.work_ad_id = w.id
                 WHERE wf.user_id = ? AND w.city_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID, cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.WorkAdFavorite
	for rows.Next() {
		var fav models.WorkAdFavorite
		var price, priceTo sql.NullFloat64
		var imagesJSON sql.NullString
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.WorkAdID, &fav.Name, &price, &priceTo, &fav.Negotiable, &fav.HidePhone, &fav.Status, &fav.CreatedAt, &imagesJSON)
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

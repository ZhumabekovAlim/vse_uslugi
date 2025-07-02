package repositories

import (
	"context"
	"database/sql"
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
	query := `SELECT id, user_id, ad_id FROM ad_favorites WHERE user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.AdFavorite
	for rows.Next() {
		var fav models.AdFavorite
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.AdID)
		if err != nil {
			return nil, err
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

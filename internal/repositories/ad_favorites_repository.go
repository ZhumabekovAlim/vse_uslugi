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
	query := `SELECT af.id, af.user_id, af.ad_id, a.name, a.price, a.price_to, a.on_site, a.negotiable, a.hide_phone, a.status, a.created_at
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
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.AdID, &fav.Name, &price, &priceTo, &fav.OnSite, &fav.Negotiable, &fav.HidePhone, &fav.Status, &fav.CreatedAt)
		if err != nil {
			return nil, err
		}
		if price.Valid {
			fav.Price = &price.Float64
		}
		if priceTo.Valid {
			fav.PriceTo = &priceTo.Float64
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

package repositories

import (
	"context"
	"database/sql"
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
	query := `SELECT sf.id, sf.user_id, sf.service_id, s.name, s.price, s.price_to, s.on_site, s.negotiable, s.hide_phone, s.status, s.created_at
             FROM service_favorites sf
             JOIN service s ON sf.service_id = s.id
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
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.ServiceID, &fav.Name, &price, &priceTo, &fav.OnSite, &fav.Negotiable, &fav.HidePhone, &fav.Status, &fav.CreatedAt)
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

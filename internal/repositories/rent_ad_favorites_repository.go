package repositories

import (
	"context"
	"database/sql"
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
	query := `SELECT rf.id, rf.user_id, rf.rent_ad_id, ra.name, ra.price, ra.status, ra.created_at
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
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.RentAdID, &fav.Name, &fav.Price, &fav.Status, &fav.CreatedAt)
		if err != nil {
			return nil, err
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

package repositories

import (
	"context"
	"database/sql"
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
	query := `SELECT id, user_id, rent_id FROM rent_favorites WHERE user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.RentFavorite
	for rows.Next() {
		var fav models.RentFavorite
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.RentID)
		if err != nil {
			return nil, err
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

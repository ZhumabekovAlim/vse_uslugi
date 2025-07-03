package repositories

import (
	"context"
	"database/sql"
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
	query := `SELECT id, user_id, work_ad_id FROM work_ad_favorites WHERE user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.WorkAdFavorite
	for rows.Next() {
		var fav models.WorkAdFavorite
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.WorkAdID)
		if err != nil {
			return nil, err
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

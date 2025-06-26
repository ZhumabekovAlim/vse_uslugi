package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type WorkFavoriteRepository struct {
	DB *sql.DB
}

func (r *WorkFavoriteRepository) AddWorkToFavorites(ctx context.Context, fav models.WorkFavorite) error {
	query := `INSERT INTO work_favorites (user_id, work_id) VALUES (?, ?)`
	_, err := r.DB.ExecContext(ctx, query, fav.UserID, fav.WorkID)
	return err
}

func (r *WorkFavoriteRepository) RemoveWorkFromFavorites(ctx context.Context, userID, workID int) error {
	query := `DELETE FROM work_favorites WHERE user_id = ? AND work_id = ?`
	_, err := r.DB.ExecContext(ctx, query, userID, workID)
	return err
}

func (r *WorkFavoriteRepository) IsWorkFavorite(ctx context.Context, userID, workID int) (bool, error) {
	query := `SELECT COUNT(*) FROM work_favorites WHERE user_id = ? AND work_id = ?`
	var count int
	err := r.DB.QueryRowContext(ctx, query, userID, workID).Scan(&count)
	return count > 0, err
}

func (r *WorkFavoriteRepository) GetWorkFavoritesByUser(ctx context.Context, userID int) ([]models.WorkFavorite, error) {
	query := `SELECT id, user_id, work_id FROM work_favorites WHERE user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []models.WorkFavorite
	for rows.Next() {
		var fav models.WorkFavorite
		err := rows.Scan(&fav.ID, &fav.UserID, &fav.WorkID)
		if err != nil {
			return nil, err
		}
		favs = append(favs, fav)
	}
	return favs, nil
}

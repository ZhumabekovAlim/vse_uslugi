package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

// LocationRepository handles persistence for user locations.
type LocationRepository struct {
	DB *sql.DB
}

// SetLocation stores the latest coordinates for a user.
func (r *LocationRepository) SetLocation(ctx context.Context, loc models.Location) error {
	query := `UPDATE users SET latitude = ?, longitude = ?, updated_at = ? WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, loc.Latitude, loc.Longitude, time.Now(), loc.UserID)
	return err
}

// GetLocation retrieves last known coordinates for a user.
func (r *LocationRepository) GetLocation(ctx context.Context, userID int) (models.Location, error) {
	var loc models.Location
	loc.UserID = userID
	query := `SELECT latitude, longitude FROM users WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&loc.Latitude, &loc.Longitude)
	return loc, err
}

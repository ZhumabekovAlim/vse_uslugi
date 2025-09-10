package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"naimuBack/internal/models"
	"strconv"
	"strings"
	"time"
)

// LocationRepository handles persistence for user locations.
type LocationRepository struct {
	DB *sql.DB
}

// SetLocation stores the latest coordinates for a user.
func (r *LocationRepository) SetLocation(ctx context.Context, loc models.Location) error {
	query := `UPDATE users SET latitude = ?, longitude = ?, is_online = ?, updated_at = ? WHERE id = ?`

	var lat, lon interface{}
	if loc.Latitude != nil {
		lat = *loc.Latitude
	}
	if loc.Longitude != nil {
		lon = *loc.Longitude
	}
	isOnline := loc.Latitude != nil && loc.Longitude != nil

	_, err := r.DB.ExecContext(ctx, query, lat, lon, isOnline, time.Now(), loc.UserID)
	return err
}

// GetLocation retrieves last known coordinates for a user.
func (r *LocationRepository) GetLocation(ctx context.Context, userID int) (models.Location, error) {
	var loc models.Location
	loc.UserID = userID
	var lat, lon sql.NullString
	query := `SELECT latitude, longitude FROM users WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, userID).Scan(&lat, &lon)
	if lat.Valid {
		if v, err2 := strconv.ParseFloat(lat.String, 64); err2 == nil {
			loc.Latitude = &v
		}
	}
	if lon.Valid {
		if v, err2 := strconv.ParseFloat(lon.String, 64); err2 == nil {
			loc.Longitude = &v
		}
	}
	return loc, err
}

// ClearLocation removes coordinates and marks user offline.
func (r *LocationRepository) ClearLocation(ctx context.Context, userID int) error {
	query := `UPDATE users SET latitude = NULL, longitude = NULL, is_online = 0, updated_at = ? WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, time.Now(), userID)
	return err
}

// GetExecutors returns executors on line with active items filtered by parameters.
func (r *LocationRepository) GetExecutors(ctx context.Context, f models.ExecutorLocationFilter) ([]models.ExecutorLocation, error) {
	tables := []string{"service", "ad", "rent", "rent_ad", "work", "work_ad"}
	var result []models.ExecutorLocation

	for _, table := range tables {
		query, args := buildExecutorQuery(table, f)
		rows, err := r.DB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var item models.ExecutorLocation
			var latStr, lonStr sql.NullString
			var avatar sql.NullString
			if err := rows.Scan(&item.UserID, &item.Name, &item.Surname, &avatar, &latStr, &lonStr, &item.ItemID, &item.ItemName, &item.Description, &item.Price, &item.AvgRating); err != nil {
				rows.Close()
				return nil, err
			}
			if avatar.Valid {
				item.Avatar = &avatar.String
			}
			if latStr.Valid {
				if v, err2 := strconv.ParseFloat(latStr.String, 64); err2 == nil {
					item.Latitude = &v
				}
			}
			if lonStr.Valid {
				if v, err2 := strconv.ParseFloat(lonStr.String, 64); err2 == nil {
					item.Longitude = &v
				}
			}
			item.Type = table
			result = append(result, item)
		}
		rows.Close()
	}
	return result, nil
}

func buildExecutorQuery(table string, f models.ExecutorLocationFilter) (string, []interface{}) {
	base := fmt.Sprintf(`SELECT u.id, u.name, u.surname, u.avatar_path, u.latitude, u.longitude, s.id, s.name, s.description, s.price, s.avg_rating FROM %s s JOIN users u ON u.id = s.user_id`, table)

	var where []string
	var args []interface{}
	where = append(where, "s.status = 'active'")
	where = append(where, "u.is_online = 1")
	where = append(where, "u.latitude IS NOT NULL")
	where = append(where, "u.longitude IS NOT NULL")

	if len(f.CategoryIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(f.CategoryIDs)), ",")
		where = append(where, fmt.Sprintf("s.category_id IN (%s)", placeholders))
		for _, id := range f.CategoryIDs {
			args = append(args, id)
		}
	}
	if len(f.SubcategoryIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(f.SubcategoryIDs)), ",")
		where = append(where, fmt.Sprintf("s.subcategory_id IN (%s)", placeholders))
		for _, id := range f.SubcategoryIDs {
			args = append(args, id)
		}
	}
	if f.PriceFrom != 0 || f.PriceTo != 0 {
		where = append(where, "s.price BETWEEN ? AND ?")
		args = append(args, f.PriceFrom, f.PriceTo)
	}
	if len(f.AvgRating) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(f.AvgRating)), ",")
		where = append(where, fmt.Sprintf("s.avg_rating IN (%s)", placeholders))
		for _, r := range f.AvgRating {
			args = append(args, r)
		}
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	return query, args
}

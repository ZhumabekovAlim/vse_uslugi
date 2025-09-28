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
func (r *LocationRepository) GetExecutors(ctx context.Context, f models.ExecutorLocationFilter) ([]models.ExecutorLocationGroup, error) {
	tables := []string{"service", "ad", "rent", "rent_ad", "work", "work_ad"}
	groups := make(map[int]*models.ExecutorLocationGroup)
	var order []int

	for _, table := range tables {
		query, args := buildExecutorQuery(table, f)
		rows, err := r.DB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var (
				userID    int
				name      string
				surname   string
				avatar    sql.NullString
				latStr    sql.NullString
				lonStr    sql.NullString
				itemID    int
				itemName  string
				desc      string
				price     float64
				avgRating float64
			)

			if err := rows.Scan(&userID, &name, &surname, &avatar, &latStr, &lonStr, &itemID, &itemName, &desc, &price, &avgRating); err != nil {
				rows.Close()
				return nil, err
			}

			group, exists := groups[userID]
			if !exists {
				group = &models.ExecutorLocationGroup{
					UserID:  userID,
					Name:    name,
					Surname: surname,
				}
				if avatar.Valid {
					val := avatar.String
					group.Avatar = &val
				}
				if lat := toFloatPtr(latStr); lat != nil {
					group.Latitude = lat
				}
				if lon := toFloatPtr(lonStr); lon != nil {
					group.Longitude = lon
				}
				groups[userID] = group
				order = append(order, userID)
			} else {
				if group.Avatar == nil && avatar.Valid {
					val := avatar.String
					group.Avatar = &val
				}
				if group.Latitude == nil {
					if lat := toFloatPtr(latStr); lat != nil {
						group.Latitude = lat
					}
				}
				if group.Longitude == nil {
					if lon := toFloatPtr(lonStr); lon != nil {
						group.Longitude = lon
					}
				}
			}

			item := models.ExecutorLocationItem{
				ID:          itemID,
				Name:        itemName,
				Description: desc,
				Price:       price,
				AvgRating:   avgRating,
			}

			switch table {
			case "service":
				group.Services = append(group.Services, item)
			case "ad":
				group.Ads = append(group.Ads, item)
			case "work_ad":
				group.WorkAds = append(group.WorkAds, item)
			case "work":
				group.Works = append(group.Works, item)
			case "rent_ad":
				group.RentAds = append(group.RentAds, item)
			case "rent":
				group.Rents = append(group.Rents, item)
			}
		}
		rows.Close()
	}

	result := make([]models.ExecutorLocationGroup, 0, len(order))
	for _, userID := range order {
		result = append(result, *groups[userID])
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

func toFloatPtr(value sql.NullString) *float64 {
	if !value.Valid {
		return nil
	}
	if v, err := strconv.ParseFloat(value.String, 64); err == nil {
		val := v
		return &val
	}
	return nil
}

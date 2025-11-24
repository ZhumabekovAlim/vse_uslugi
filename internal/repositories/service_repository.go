package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"naimuBack/internal/models"
	"sort"
	"strings"
	"time"
)

var (
	ErrServiceNotFound = errors.New("service not found")
)

type ServiceRepository struct {
	DB *sql.DB
}

func (r *ServiceRepository) CreateService(ctx context.Context, service models.Service) (models.Service, error) {
	query := `
        INSERT INTO service (name, address, price, user_id, images, videos, category_id, subcategory_id, description, avg_rating, top, liked, status, latitude, longitude, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                `
	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(service.Images)
	if err != nil {
		return models.Service{}, err
	}

	videosJSON, err := json.Marshal(service.Videos)
	if err != nil {
		return models.Service{}, err
	}

	var latitude interface{}
	if service.Latitude != nil && *service.Latitude != "" {
		latitude = *service.Latitude
	}

	var longitude interface{}
	if service.Longitude != nil && *service.Longitude != "" {
		longitude = *service.Longitude
	}

	var subcategory interface{}
	if service.SubcategoryID != 0 {
		subcategory = service.SubcategoryID
	}

	result, err := r.DB.ExecContext(ctx, query,
		service.Name,
		service.Address,
		service.Price,
		service.UserID,
		string(imagesJSON),
		string(videosJSON),
		service.CategoryID,
		subcategory,
		service.Description,
		service.AvgRating,
		service.Top,
		service.Liked,
		service.Status,
		latitude,
		longitude,
		service.CreatedAt,
	)
	if err != nil {
		return models.Service{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.Service{}, err
	}
	service.ID = int(lastID)
	return service, nil
}

func (r *ServiceRepository) GetServiceByID(ctx context.Context, id int, userID int) (models.Service, error) {
	query := `
         SELECT s.id, s.name, s.address, s.price, s.user_id,
                u.id, u.name, u.surname, COALESCE(u.review_rating, 0), u.avatar_path,
                  CASE WHEN sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                  s.images, s.videos, s.category_id, c.name, s.subcategory_id, sub.name, sub.name_kz,
                  s.description, s.avg_rating, s.top, s.liked,
                  CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                  s.latitude, s.longitude, s.status, s.created_at, s.updated_at
           FROM service s
           JOIN users u ON s.user_id = u.id
           JOIN categories c ON s.category_id = c.id
           LEFT JOIN subcategories sub ON s.subcategory_id = sub.id
           LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
           WHERE s.id = ?
   `

	var s models.Service
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var respondedStr string
	var subcategoryID sql.NullInt64
	var subcategoryName, subcategoryNameKz sql.NullString
	var status, description, top sql.NullString
	var avgRating sql.NullFloat64
	var liked sql.NullBool

	err := r.DB.QueryRowContext(ctx, query, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &subcategoryID, &subcategoryName, &subcategoryNameKz,
		&description, &avgRating, &top, &liked, &respondedStr,
		&lat, &lon, &status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, ErrServiceNotFound
	}
	if err != nil {
		return models.Service{}, err
	}

	if status.Valid {
		s.Status = status.String
	}

	if description.Valid {
		s.Description = description.String
	}

	if avgRating.Valid {
		s.AvgRating = avgRating.Float64
	}

	if top.Valid {
		s.Top = top.String
	}

	if liked.Valid {
		s.Liked = liked.Bool
	}

	if subcategoryID.Valid {
		s.SubcategoryID = int(subcategoryID.Int64)
	}
	if subcategoryName.Valid {
		s.SubcategoryName = subcategoryName.String
	}
	if subcategoryNameKz.Valid {
		s.SubcategoryNameKz = subcategoryNameKz.String
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Service{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Service{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Responded = respondedStr == "1"

	if lat.Valid {
		s.Latitude = &lat.String
	}
	if lon.Valid {
		s.Longitude = &lon.String
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *ServiceRepository) UpdateService(ctx context.Context, service models.Service) (models.Service, error) {
	query := `
        UPDATE service
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, latitude = ?, longitude = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(service.Images)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	videosJSON, err := json.Marshal(service.Videos)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to marshal videos: %w", err)
	}
	updatedAt := time.Now()
	service.UpdatedAt = &updatedAt
	var latitude interface{}
	if service.Latitude != nil && *service.Latitude != "" {
		latitude = *service.Latitude
	}

	var longitude interface{}
	if service.Longitude != nil && *service.Longitude != "" {
		longitude = *service.Longitude
	}

	result, err := r.DB.ExecContext(ctx, query,
		service.Name, service.Address, service.Price, service.UserID, imagesJSON, videosJSON,
		service.CategoryID, service.SubcategoryID, service.Description, service.AvgRating, service.Top, service.Liked, service.Status, latitude, longitude, service.UpdatedAt, service.ID,
	)
	if err != nil {
		return models.Service{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Service{}, err
	}
	if rowsAffected == 0 {
		return models.Service{}, ErrServiceNotFound
	}
	return r.GetServiceByID(ctx, service.ID, 0)
}

func (r *ServiceRepository) DeleteService(ctx context.Context, id int) error {
	query := `DELETE FROM service WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrServiceNotFound
	}
	return nil
}

func (r *ServiceRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE service SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrServiceNotFound
	}
	return nil
}
func (r *ServiceRepository) GetServicesWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int) ([]models.Service, float64, float64, error) {
	var (
		services   []models.Service
		params     []interface{}
		conditions []string
	)

	conditions = append(conditions, "s.status = 'active'")

	baseQuery := `
           SELECT s.id, s.name, s.address, s.price, s.user_id,
                  u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                     s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top,
                     s.latitude, s.longitude,

             CASE WHEN sf.service_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

                     s.status,  s.created_at, s.updated_at
              FROM service s
              LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
              JOIN users u ON s.user_id = u.id
              INNER JOIN categories c ON s.category_id = c.id

      `
	params = append(params, userID)

	if cityID > 0 {
		conditions = append(conditions, "u.city_id = ?")
		params = append(params, cityID)
	}

	// Filters
	if len(categories) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(categories)), ",")
		conditions = append(conditions, fmt.Sprintf("s.category_id IN (%s)", placeholders))
		for _, cat := range categories {
			params = append(params, cat)
		}
	}

	if len(subcategories) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(subcategories)), ",")
		conditions = append(conditions, fmt.Sprintf("s.subcategory_id IN (%s)", placeholders))
		for _, sub := range subcategories {
			params = append(params, sub)
		}
	}

	if priceFrom > 0 {
		conditions = append(conditions, "s.price >= ?")
		params = append(params, priceFrom)
	}
	if priceTo > 0 {
		conditions = append(conditions, "s.price <= ?")
		params = append(params, priceTo)
	}

	if len(ratings) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ratings)), ",")
		conditions = append(conditions, fmt.Sprintf("s.avg_rating IN (%s)", placeholders))
		for _, r := range ratings {
			params = append(params, r)
		}
	}

	// Final WHERE clause
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Sorting
	switch sortOption {
	case 1:
		baseQuery += ` ORDER BY ( SELECT COUNT(*) FROM reviews r WHERE r.service_id = s.id) DESC `

	case 2:
		baseQuery += ` ORDER BY s.price ASC`
	case 3:
		baseQuery += ` ORDER BY s.price DESC`
	default:
		baseQuery += ` ORDER BY s.created_at DESC`
	}

	// Pagination
	baseQuery += " LIMIT ? OFFSET ?"
	params = append(params, limit, offset)

	// Query
	rows, err := r.DB.QueryContext(ctx, baseQuery, params...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.Service
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		var likedStr string
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &lat, &lon, &likedStr, &s.Status,

			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if lat.Valid {
			s.Latitude = &lat.String
		}
		if lon.Valid {
			s.Longitude = &lon.String
		}

		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
		}

		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode videos error: %w", err)
			}
		}

		s.Liked = likedStr == "1"

		s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}

		services = append(services, s)
	}

	sortServicesByTop(services)

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM service`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return services, 0, 0, nil // fallback
	}

	return services, minPrice, maxPrice, nil
}

func (r *ServiceRepository) GetServicesByUserID(ctx context.Context, userID int) ([]models.Service, error) {
	query := `
           SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, s.liked, s.status, s.latitude, s.longitude, s.created_at, s.updated_at
              FROM service s
              JOIN users u ON s.user_id = u.id
              WHERE user_id = ?
       `

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &lat, &lon, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(imagesJSON) > 0 {
			if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		if lat.Valid {
			s.Latitude = &lat.String
		}
		if lon.Valid {
			s.Longitude = &lon.String
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)

		services = append(services, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortServicesByTop(services)

	return services, nil
}

func (r *ServiceRepository) GetFilteredServicesPost(ctx context.Context, req models.FilterServicesRequest) ([]models.FilteredService, error) {
	query := `
SELECT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at
FROM service s
JOIN users u ON s.user_id = u.id
WHERE 1=1
`
	args := []interface{}{}

	query += " AND s.status = 'active'"

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
	}

	if req.CityID > 0 {
		query += " AND u.city_id = ?"
		args = append(args, req.CityID)
	}

	// Category
	if len(req.CategoryIDs) > 0 {
		placeholders := strings.Repeat("?,", len(req.CategoryIDs))
		placeholders = placeholders[:len(placeholders)-1] // remove trailing comma
		query += fmt.Sprintf(" AND s.category_id IN (%s)", placeholders)
		for _, id := range req.CategoryIDs {
			args = append(args, id)
		}
	}

	// Subcategory
	if len(req.SubcategoryIDs) > 0 {
		placeholders := strings.Repeat("?,", len(req.SubcategoryIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.subcategory_id IN (%s)", placeholders)
		for _, id := range req.SubcategoryIDs {
			args = append(args, id)
		}
	}

	// Ratings
	if len(req.AvgRatings) > 0 {
		sort.Ints(req.AvgRatings)
		query += " AND s.avg_rating >= ?"
		args = append(args, float64(req.AvgRatings[0]))
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += " ORDER BY (SELECT COUNT(*) FROM reviews r WHERE r.service_id = s.id) DESC"
	case 2:
		query += " ORDER BY s.price DESC"
	case 3:
		query += " ORDER BY s.price ASC"
	default:
		query += " ORDER BY s.created_at DESC"
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.FilteredService
	for rows.Next() {
		var s models.FilteredService
		var lat, lon sql.NullString
		var imagesJSON, videosJSON []byte
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.ServiceID, &s.ServiceName, &s.ServiceAddress, &s.ServicePrice, &s.ServiceDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if lat.Valid {
			latVal := lat.String
			s.ServiceLatitude = &latVal
		}
		if lon.Valid {
			lonVal := lon.String
			s.ServiceLongitude = &lonVal
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		services = append(services, s)
	}

	sortFilteredServicesByTop(services)
	return services, nil
}

func (r *ServiceRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Service, error) {
	query := `
     SELECT
             s.id, s.name, s.address, s.price, s.user_id,
             u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                s.images, s.videos, s.category_id, s.subcategory_id, s.description,
                s.avg_rating, s.top, s.liked, s.status, s.latitude, s.longitude,
                s.created_at, s.updated_at
        FROM service s
        JOIN users u ON s.user_id = u.id
        WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &lat, &lon,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("json decode error: %w", err)
		}
		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}
		if lat.Valid {
			s.Latitude = &lat.String
		}
		if lon.Valid {
			s.Longitude = &lon.String
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)
		services = append(services, s)
	}
	sortServicesByTop(services)
	return services, nil
}

func (r *ServiceRepository) GetFilteredServicesWithLikes(ctx context.Context, req models.FilterServicesRequest, userID int) ([]models.FilteredService, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
SELECT DISTINCT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at,
CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
FROM service s
JOIN users u ON s.user_id = u.id
LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
WHERE 1=1
`

	args := []interface{}{userID, userID}

	query += " AND s.status = 'active'"

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
	}

	if req.CityID > 0 {
		query += " AND u.city_id = ?"
		args = append(args, req.CityID)
	}

	// Category filter
	if len(req.CategoryIDs) > 0 {
		placeholders := strings.Repeat("?,", len(req.CategoryIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.category_id IN (%s)", placeholders)
		for _, id := range req.CategoryIDs {
			args = append(args, id)
		}
	}

	// Subcategory filter
	if len(req.SubcategoryIDs) > 0 {
		placeholders := strings.Repeat("?,", len(req.SubcategoryIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.subcategory_id IN (%s)", placeholders)
		for _, id := range req.SubcategoryIDs {
			args = append(args, id)
		}
	}

	// Ratings filter
	if len(req.AvgRatings) > 0 {
		sort.Ints(req.AvgRatings)
		query += " AND s.avg_rating >= ?"
		args = append(args, float64(req.AvgRatings[0]))
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += " ORDER BY (SELECT COUNT(*) FROM reviews r WHERE r.service_id = s.id) DESC"
	case 2:
		query += " ORDER BY s.price DESC"
	case 3:
		query += " ORDER BY s.price ASC"
	default:
		query += " ORDER BY s.created_at DESC"
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("[ERROR] Query execution failed: %v", err)
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("[WARN] Failed to close rows: %v", cerr)
		}
	}()

	var services []models.FilteredService
	for rows.Next() {
		var s models.FilteredService
		var lat, lon sql.NullString
		var imagesJSON, videosJSON []byte
		var likedStr, respondedStr string
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.ServiceID, &s.ServiceName, &s.ServiceAddress, &s.ServicePrice, &s.ServiceDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if lat.Valid {
			latVal := lat.String
			s.ServiceLatitude = &latVal
		}
		if lon.Valid {
			lonVal := lon.String
			s.ServiceLongitude = &lonVal
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}
		s.Liked = likedStr == "1"
		s.Responded = respondedStr == "1"
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		services = append(services, s)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error after reading rows: %v", err)
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	sortFilteredServicesByTop(services)
	log.Printf("[INFO] Successfully fetched %d services", len(services))
	return services, nil
}

func (r *ServiceRepository) GetServiceByServiceIDAndUserID(ctx context.Context, serviceID int, userID int) (models.Service, error) {
	query := `
            SELECT
                    s.id, s.name, s.address, s.price, s.user_id,
                    u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                       CASE WHEN sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                       s.images, s.videos, s.category_id, c.name,
                       s.subcategory_id, sub.name, sub.name_kz,
                       s.description, s.avg_rating, s.top,
                       CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
                       CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                       sc.chat_id,
                       cch.user1_id, cu1.name, cu1.surname, cu1.avatar_path,
                       cch.user2_id, cu2.name, cu2.surname, cu2.avatar_path,
                       cch.created_at,
                       s.latitude, s.longitude, s.status, s.created_at, s.updated_at
               FROM service s
               JOIN users u ON s.user_id = u.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
               LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
               LEFT JOIN service_confirmations sc ON sc.service_id = s.id AND (sc.client_id = ? OR sc.performer_id = ?)
               LEFT JOIN chats cch ON cch.id = sc.chat_id
               LEFT JOIN users cu1 ON cu1.id = cch.user1_id
               LEFT JOIN users cu2 ON cu2.id = cch.user2_id
               WHERE s.id = ?
       `

	var s models.Service
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString

	var likedStr, respondedStr string
	var chatID, chatUser1ID, chatUser2ID sql.NullInt64
	var chatUser1Name, chatUser1Surname, chatUser2Name, chatUser2Surname sql.NullString
	var chatUser1Avatar, chatUser2Avatar sql.NullString
	var chatCreatedAt sql.NullTime

	err := r.DB.QueryRowContext(ctx, query, userID, userID, userID, userID, serviceID).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.Description, &s.AvgRating, &s.Top,
		&likedStr, &respondedStr,
		&chatID,
		&chatUser1ID, &chatUser1Name, &chatUser1Surname, &chatUser1Avatar,
		&chatUser2ID, &chatUser2Name, &chatUser2Surname, &chatUser2Avatar,
		&chatCreatedAt,
		&lat, &lon, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, errors.New("service not found")
	}
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Service{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Service{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	if lat.Valid {
		s.Latitude = &lat.String
	}
	if lon.Valid {
		s.Longitude = &lon.String
	}

	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"

	if chatID.Valid {
		s.Chat = &models.Chat{ID: int(chatID.Int64)}

		if chatUser1ID.Valid {
			s.Chat.User1ID = int(chatUser1ID.Int64)
			s.Chat.User1.Name = chatUser1Name.String
			s.Chat.User1.Surname = chatUser1Surname.String
			if chatUser1Avatar.Valid {
				s.Chat.User1.AvatarPath = &chatUser1Avatar.String
			}
		}

		if chatUser2ID.Valid {
			s.Chat.User2ID = int(chatUser2ID.Int64)
			s.Chat.User2.Name = chatUser2Name.String
			s.Chat.User2.Surname = chatUser2Surname.String
			if chatUser2Avatar.Valid {
				s.Chat.User2.AvatarPath = &chatUser2Avatar.String
			}
		}

		if chatCreatedAt.Valid {
			s.Chat.CreatedAt = chatCreatedAt.Time
		}
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

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
    INSERT INTO service (name, address, on_site, price, price_to, negotiable, hide_phone, user_id, city_id, images, videos, category_id, subcategory_id, work_time_from, work_time_to, description, avg_rating, top, liked, status, latitude, longitude, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

	var price interface{}
	if service.Price != nil {
		price = *service.Price
	}

	var priceTo interface{}
	if service.PriceTo != nil {
		priceTo = *service.PriceTo
	}

	var cityID interface{}
	if service.CityID != 0 {
		cityID = service.CityID
	}

	result, err := r.DB.ExecContext(ctx, query,
		service.Name,
		service.Address,
		service.OnSite,
		price,
		priceTo,
		service.Negotiable,
		service.HidePhone,
		service.UserID,
		cityID,
		string(imagesJSON),
		string(videosJSON),
		service.CategoryID,
		subcategory,
		service.WorkTimeFrom,
		service.WorkTimeTo,
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
         SELECT s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id, s.city_id, city.name,
                u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                  CASE WHEN s.hide_phone = 0 AND sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                  s.images, s.videos, s.category_id, c.name, s.subcategory_id, sub.name, sub.name_kz,
                  s.work_time_from, s.work_time_to, s.description, s.avg_rating, s.top, s.negotiable, s.hide_phone,
                  CASE WHEN sf.service_id IS NOT NULL THEN '1' ELSE '0' END AS liked,
                  CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                  s.latitude, s.longitude, s.status, s.created_at, s.updated_at
               FROM service s
               JOIN users u ON s.user_id = u.id
               LEFT JOIN cities city ON s.city_id = city.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
               LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
               WHERE s.id = ?
       `

	var s models.Service
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var price, priceTo sql.NullFloat64
	var respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &respondedStr,
		&lat, &lon, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, errors.New("not found")
	}
	if err != nil {
		return models.Service{}, err
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

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
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

func (r *ServiceRepository) GetServiceByIDWithCity(ctx context.Context, id int, userID int, cityID int) (models.Service, error) {
	query := `
         SELECT s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id, s.city_id, city.name,
                u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                  CASE WHEN s.hide_phone = 0 AND sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                  s.images, s.videos, s.category_id, c.name, s.subcategory_id, sub.name, sub.name_kz,
                  s.work_time_from, s.work_time_to, s.description, s.avg_rating, s.top, s.negotiable, s.hide_phone, s.liked,
                  CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                  s.latitude, s.longitude, s.status, s.created_at, s.updated_at
               FROM service s
               JOIN users u ON s.user_id = u.id
               LEFT JOIN cities city ON s.city_id = city.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
               WHERE s.id = ? AND s.city_id = ?
       `

	var s models.Service
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var price, priceTo sql.NullFloat64
	var respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, id, cityID).Scan(
		&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &respondedStr,
		&lat, &lon, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, errors.New("not found")
	}
	if err != nil {
		return models.Service{}, err
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

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
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
    SET name = ?, address = ?, on_site = ?, price = ?, price_to = ?, negotiable = ?, hide_phone = ?, user_id = ?, city_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
        work_time_from = ?, work_time_to = ?, description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, latitude = ?, longitude = ?, updated_at = ?
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

	var price interface{}
	if service.Price != nil {
		price = *service.Price
	}
	var priceTo interface{}
	if service.PriceTo != nil {
		priceTo = *service.PriceTo
	}

	var cityID interface{}
	if service.CityID != 0 {
		cityID = service.CityID
	}

	result, err := r.DB.ExecContext(ctx, query,
		service.Name, service.Address, service.OnSite, price, priceTo, service.Negotiable, service.HidePhone, service.UserID, cityID, imagesJSON, videosJSON,
		service.CategoryID, service.SubcategoryID, service.WorkTimeFrom, service.WorkTimeTo, service.Description, service.AvgRating, service.Top, service.Liked, service.Status, latitude, longitude, service.UpdatedAt, service.ID,
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

func (r *ServiceRepository) ArchiveByUserID(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE service SET status = 'archive', updated_at = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}
func (r *ServiceRepository) GetServicesWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int, onSite, negotiable *bool) ([]models.Service, float64, float64, error) {
	var (
		services   []models.Service
		params     []interface{}
		conditions []string
	)

	baseQuery := `
          SELECT s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id, city.name,
                 u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                    s.images, s.videos, s.category_id, s.subcategory_id, s.description,
                    s.work_time_from, s.work_time_to, s.avg_rating, s.top, s.negotiable, s.hide_phone,
                    s.latitude, s.longitude,

             CASE WHEN sf.service_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

                     s.status,  s.created_at, s.updated_at
              FROM service s
              LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
              JOIN users u ON s.user_id = u.id
              LEFT JOIN cities city ON s.city_id = city.id
              INNER JOIN categories c ON s.category_id = c.id

      `
	params = append(params, userID)

	conditions = append(conditions, "s.status != 'archive'")

	if cityID > 0 {
		conditions = append(conditions, "s.city_id = ?")
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

	if onSite != nil {
		conditions = append(conditions, "s.on_site = ?")
		params = append(params, *onSite)
	}

	if negotiable != nil {
		conditions = append(conditions, "s.negotiable = ?")
		params = append(params, *negotiable)
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
		baseQuery += ` ORDER BY s.created_at ASC`

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
		var price, priceTo sql.NullFloat64
		var likedStr string
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID, &s.CityName,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.WorkTimeFrom, &s.WorkTimeTo, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &lat, &lon, &likedStr, &s.Status,

			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
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

	liftListingsTopOnly(services, func(s models.Service) string { return s.Top })

	// Get min/max prices
	var minPrice, maxPrice sql.NullFloat64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM service`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return services, 0, 0, nil // fallback
	}

	return services, minPrice.Float64, maxPrice.Float64, nil
}

func (r *ServiceRepository) GetServicesByUserID(ctx context.Context, userID int) ([]models.Service, error) {
	query := `
       SELECT s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id, s.city_id, city.name, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.work_time_from, s.work_time_to, s.avg_rating, s.top, s.negotiable, s.hide_phone, CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.status, s.latitude, s.longitude, s.created_at, s.updated_at
          FROM service s
          JOIN users u ON s.user_id = u.id
          LEFT JOIN cities city ON s.city_id = city.id
          LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
          WHERE s.user_id = ?
   `

	rows, err := r.DB.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		var imagesJSON []byte
		var videosJSON []byte
		var likedStr string
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.WorkTimeFrom, &s.WorkTimeTo, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &likedStr, &s.Status, &lat, &lon, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
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

		s.Liked = likedStr == "1"
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

s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.negotiable, s.hide_phone, s.description,
s.work_time_from, s.work_time_to, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at
FROM service s
JOIN users u ON s.user_id = u.id
WHERE 1=1 AND s.status != 'archive'
`
	args := []interface{}{}

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

	if req.Negotiable != nil {
		query += " AND s.negotiable = ?"
		args = append(args, *req.Negotiable)
	}

	if req.OnSite != nil {
		query += " AND s.on_site = ?"
		args = append(args, *req.OnSite)
	}

	if req.TwentyFourSeven {
		query += " AND ((s.work_time_from = '00:00' AND s.work_time_to = '23:59') OR (s.work_time_from = '00:00:00' AND s.work_time_to = '23:59:59'))"
	}

	if req.OpenNow {
		query += " AND TIME(NOW()) BETWEEN TIME(s.work_time_from) AND TIME(s.work_time_to)"
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += " ORDER BY s.created_at ASC"
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
		var price, priceTo sql.NullFloat64
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.ServiceID, &s.ServiceName, &s.ServiceAddress, &s.ServiceOnSite, &price, &priceTo, &s.ServiceNegotiable, &s.ServiceHidePhone, &s.ServiceDescription, &s.WorkTimeFrom, &s.WorkTimeTo, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if price.Valid {
			s.ServicePrice = &price.Float64
		}
		if priceTo.Valid {
			s.ServicePriceTo = &priceTo.Float64
		}
		if lat.Valid {
			latVal := lat.String
			s.ServiceLatitude = &latVal
		}
		if lon.Valid {
			lonVal := lon.String
			s.ServiceLongitude = &lonVal
		}

		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, s.ServiceLatitude, s.ServiceLongitude)
		if req.RadiusKm != nil && (s.Distance == nil || *s.Distance > *req.RadiusKm) {
			continue
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

	liftListingsTopOnly(services, func(s models.FilteredService) string { return s.Top })
	return services, nil
}

func (r *ServiceRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Service, error) {
	query := `
     SELECT
            s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id,
            u.id, u.name, u.surname, u.review_rating, u.avatar_path,
            s.images, s.videos, s.category_id, s.subcategory_id, s.description,
            s.work_time_from, s.work_time_to, s.avg_rating, s.top, s.negotiable, s.hide_phone, s.liked, s.status, s.latitude, s.longitude,
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
		var price, priceTo sql.NullFloat64
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.WorkTimeFrom, &s.WorkTimeTo, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &s.Status, &lat, &lon,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
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

s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.negotiable, s.hide_phone, s.description,
s.work_time_from, s.work_time_to, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at,
CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
FROM service s
JOIN users u ON s.user_id = u.id
LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
WHERE 1=1 AND s.status != 'archive'
`

	args := []interface{}{userID, userID}

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

	if req.Negotiable != nil {
		query += " AND s.negotiable = ?"
		args = append(args, *req.Negotiable)
	}

	if req.OnSite != nil {
		query += " AND s.on_site = ?"
		args = append(args, *req.OnSite)
	}

	if req.TwentyFourSeven {
		query += " AND ((s.work_time_from = '00:00' AND s.work_time_to = '23:59') OR (s.work_time_from = '00:00:00' AND s.work_time_to = '23:59:59'))"
	}

	if req.OpenNow {
		query += " AND TIME(NOW()) BETWEEN TIME(s.work_time_from) AND TIME(s.work_time_to)"
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += " ORDER BY s.created_at ASC"
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
		var price, priceTo sql.NullFloat64
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.ServiceID, &s.ServiceName, &s.ServiceAddress, &s.ServiceOnSite, &price, &priceTo, &s.ServiceNegotiable, &s.ServiceHidePhone, &s.ServiceDescription, &s.WorkTimeFrom, &s.WorkTimeTo, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if price.Valid {
			s.ServicePrice = &price.Float64
		}
		if priceTo.Valid {
			s.ServicePriceTo = &priceTo.Float64
		}
		if lat.Valid {
			latVal := lat.String
			s.ServiceLatitude = &latVal
		}
		if lon.Valid {
			lonVal := lon.String
			s.ServiceLongitude = &lonVal
		}

		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, s.ServiceLatitude, s.ServiceLongitude)
		if req.RadiusKm != nil && (s.Distance == nil || *s.Distance > *req.RadiusKm) {
			continue
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

	liftListingsTopOnly(services, func(s models.FilteredService) string { return s.Top })
	log.Printf("[INFO] Successfully fetched %d services", len(services))
	return services, nil
}

func (r *ServiceRepository) GetServiceByServiceIDAndUserID(ctx context.Context, serviceID int, userID int) (models.Service, error) {
	query := `
            SELECT
                    s.id, s.name, s.address, s.on_site, s.price, s.price_to, s.user_id, s.city_id,
                    u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                       CASE WHEN s.hide_phone = 0 AND sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                       s.images, s.videos, s.category_id, c.name,
                       s.subcategory_id, sub.name, sub.name_kz,
                       s.description, s.work_time_from, s.work_time_to, s.avg_rating, s.top, s.negotiable, s.hide_phone,
                       CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
                       CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                       s.latitude, s.longitude, s.status, s.created_at, s.updated_at
               FROM service s
               JOIN users u ON s.user_id = u.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
               LEFT JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = ?
               WHERE s.id = ?
       `

	var s models.Service
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString

	var likedStr, respondedStr string
	var price, priceTo sql.NullFloat64

	err := r.DB.QueryRowContext(ctx, query, userID, userID, serviceID).Scan(
		&s.ID, &s.Name, &s.Address, &s.OnSite, &price, &priceTo, &s.UserID, &s.CityID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.Description, &s.WorkTimeFrom, &s.WorkTimeTo, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone,
		&likedStr, &respondedStr, &lat, &lon, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, errors.New("service not found")
	}
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to get service: %w", err)
	}

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
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

	s.AvgRating = getAverageRating(ctx, r.DB, "reviews", "service_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

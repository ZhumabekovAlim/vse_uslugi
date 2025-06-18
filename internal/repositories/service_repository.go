package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
        INSERT INTO service (name, address, price, user_id, images, category_id, subcategory_id, description, avg_rating, top, liked, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(service.Images)
	if err != nil {
		return models.Service{}, err
	}

	result, err := r.DB.ExecContext(ctx, query,
		service.Name,
		service.Address,
		service.Price,
		service.UserID,
		string(imagesJSON),
		service.CategoryID,
		service.SubcategoryID,
		service.Description,
		service.AvgRating,
		service.Top,
		service.Liked,
		service.Status,
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

func (r *ServiceRepository) GetServiceByID(ctx context.Context, id int) (models.Service, error) {
	query := `
		SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.subcategory_id,s.description, s.avg_rating, s.top, s.liked, s.status, s.created_at, s.updated_at
		FROM service s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ?
	`

	var s models.Service
	var imagesJSON []byte
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating,
		&imagesJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
		&s.CreatedAt, &s.UpdatedAt,
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
	return s, nil
}

func (r *ServiceRepository) UpdateService(ctx context.Context, service models.Service) (models.Service, error) {
	query := `
        UPDATE service
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, category_id = ?, subcategory_id = ?, 
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(service.Images)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	updatedAt := time.Now()
	service.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		service.Name, service.Address, service.Price, service.UserID, imagesJSON,
		service.CategoryID, service.SubcategoryID, service.Description, service.AvgRating, service.Top, service.Liked, service.Status, service.UpdatedAt, service.ID,
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
	return r.GetServiceByID(ctx, service.ID)
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
func (r *ServiceRepository) GetServicesWithFilters(ctx context.Context, userID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int) ([]models.Service, float64, float64, error) {
	var (
		services   []models.Service
		params     []interface{}
		conditions []string
	)

	baseQuery := `
		SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, CASE WHEN sf.service_id IS NOT NULL THEN 'true' ELSE 'false' END AS liked, s.status,  s.created_at, s.updated_at
		FROM service s
		LEFT JOIN service_favorites sf ON sf.service_id = s.id AND sf.user_id = ?
		JOIN users u ON s.user_id = u.id
		INNER JOIN categories c ON s.category_id = c.id
		
	`
	params = append(params, userID)

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
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
		}

		if err != nil {
			return nil, 0, 0, err
		}
		services = append(services, s)
	}

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
		SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, s.liked, s.status, s.created_at, s.updated_at
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
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating, &imagesJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(imagesJSON) > 0 {
			if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		services = append(services, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return services, nil
}

func (r *ServiceRepository) GetFilteredServicesPost(ctx context.Context, req models.FilterServicesRequest) ([]models.FilteredService, error) {
	query := `
		SELECT 
			u.id, u.name, u.review_rating,
			s.id, s.name, s.price, s.description
		FROM service s
		JOIN users u ON s.user_id = u.id
		WHERE s.price BETWEEN ? AND ?
	`
	args := []interface{}{req.PriceFrom, req.PriceTo}

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
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserRating,
			&s.ServiceID, &s.ServiceName, &s.ServicePrice, &s.ServiceDescription,
		); err != nil {
			return nil, err
		}
		services = append(services, s)
	}

	return services, nil
}

func (r *ServiceRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Service, error) {
	query := `SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.subcategory_id,s.description, s.avg_rating, s.top, s.liked, s.status, s.created_at, s.updated_at
		FROM service s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ?`

	rows, err := r.DB.QueryContext(ctx, query, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	var imagesJSON []byte
	for rows.Next() {
		var s models.Service
		err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
			&s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

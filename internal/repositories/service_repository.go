package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"naimuBack/internal/models"
	"strings"
	"time"
)

var (
	ErrServiceNotFound = errors.New("service not found")
)

type ServiceRepository struct {
	DB *sql.DB
}

func (r *ServiceRepository) CreateService(ctx context.Context, s models.Service) (models.Service, error) {
	query := `
		INSERT INTO service 
			(name, address, price, user_id, images, category_id, description, avg_rating, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.DB.ExecContext(ctx, query,
		s.Name, s.Address, s.Price, s.UserID,
		s.Images, s.CategoryID, s.Description, s.AvgRating,
	)
	if err != nil {
		return models.Service{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Service{}, err
	}
	s.ID = int(id)
	return s, nil
}

func (r *ServiceRepository) GetServiceByID(ctx context.Context, id int) (models.Service, error) {
	query := `
		SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.description, s.avg_rating, s.created_at, s.updated_at
		FROM service s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ?
	`

	var s models.Service
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating,
		&s.Images, &s.CategoryID, &s.Description, &s.AvgRating,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Service{}, errors.New("not found")
	}
	if err != nil {
		return models.Service{}, err
	}

	return s, nil
}

func (r *ServiceRepository) UpdateService(ctx context.Context, service models.Service) (models.Service, error) {
	query := `
        UPDATE service
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, category_id = ?,
            description = ?, avg_rating = ?, updated_at = ?
        WHERE id = ?
    `
	updatedAt := time.Now()
	service.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		service.Name, service.Address, service.Price, service.UserID, service.Images,
		service.CategoryID, service.Description, service.AvgRating, service.UpdatedAt, service.ID,
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
func (r *ServiceRepository) GetServicesWithFilters(
	ctx context.Context,
	userID int,
	categories []int,
	subcategories []string,
	priceFrom, priceTo float64,
	ratings []float64,
	sortOption int,
	limit, offset int,
) ([]models.Service, float64, float64, error) {
	var (
		services   []models.Service
		params     []interface{}
		conditions []string
	)

	baseQuery := `
		SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, s.images, s.category_id, s.description, s.avg_rating, CASE WHEN sf.service_id IS NOT NULL THEN 'true' ELSE 'false' END AS liked,  s.created_at, s.updated_at
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
		for _, sub := range subcategories {
			conditions = append(conditions, "c.subcategories LIKE ?")
			params = append(params, "%"+sub+"%")
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
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating,
			&s.Images, &s.CategoryID, &s.Description, &s.AvgRating, &s.Liked,
			&s.CreatedAt, &s.UpdatedAt,
		)
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
		SELECT id, name, address, price, user_id, images, category_id, description, avg_rating, created_at, updated_at
		FROM service
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
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.Images,
			&s.CategoryID, &s.Description, &s.AvgRating, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		services = append(services, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return services, nil
}

func (r *ServiceRepository) FilterServices(ctx context.Context, req models.FilteredServiceRequest) ([]models.FilteredServiceResponse, error) {
	var (
		services []models.FilteredServiceResponse
		params   []interface{}
		conds    []string
	)

	baseQuery := `
	SELECT 
		s.id as service_id, s.name as service_name, s.price, s.description,
		u.id as client_id, u.name as client_name, u.review_rating
	FROM service s
	INNER JOIN users u ON u.id = s.user_id
	INNER JOIN categories c ON c.id = s.category_id
	`

	// Categories
	var categoryIDs []int
	var subcats []string
	for _, cat := range req.Categories {
		categoryIDs = append(categoryIDs, cat.ID)
		for _, sub := range cat.Subcategories {
			subcats = append(subcats, sub.Name)
		}
	}

	if len(categoryIDs) > 0 {
		q := strings.TrimSuffix(strings.Repeat("?,", len(categoryIDs)), ",")
		conds = append(conds, fmt.Sprintf("s.category_id IN (%s)", q))
		for _, id := range categoryIDs {
			params = append(params, id)
		}
	}

	// Subcategories
	if len(subcats) > 0 {
		for _, sub := range subcats {
			conds = append(conds, "c.subcategories LIKE ?")
			params = append(params, "%"+sub+"%")
		}
	}

	if req.PriceFrom > 0 {
		conds = append(conds, "s.price >= ?")
		params = append(params, req.PriceFrom)
	}
	if req.PriceTo > 0 {
		conds = append(conds, "s.price <= ?")
		params = append(params, req.PriceTo)
	}

	if len(req.Ratings) > 0 {
		q := strings.TrimSuffix(strings.Repeat("?,", len(req.Ratings)), ",")
		conds = append(conds, fmt.Sprintf("s.avg_rating IN (%s)", q))
		for _, r := range req.Ratings {
			params = append(params, float64(r))
		}
	}

	if len(conds) > 0 {
		baseQuery += " WHERE " + strings.Join(conds, " AND ")
	}

	// Sorting
	switch req.Sorting {
	case 1:
		baseQuery += " ORDER BY (SELECT COUNT(*) FROM reviews r WHERE r.service_id = s.id) DESC"
	case 2:
		baseQuery += " ORDER BY s.price ASC"
	case 3:
		baseQuery += " ORDER BY s.price DESC"
	default:
		baseQuery += " ORDER BY s.created_at DESC"
	}

	rows, err := r.DB.QueryContext(ctx, baseQuery, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.FilteredServiceResponse
		if err := rows.Scan(&s.ServiceID, &s.ServiceName, &s.ServicePrice, &s.ServiceDescription, &s.ClientID, &s.ClientName, &s.ClientRating); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *ServiceRepository) GetServicesPost(ctx context.Context, req models.GetServicesPostRequest) ([]models.GetServicesPostResponse, error) {
	var (
		services   []models.GetServicesPostResponse
		conditions []string
		params     []interface{}
	)

	query := `
	SELECT 
		u.id as client_id, u.name as client_name, u.review_rating as client_rating,
		s.id as service_id, s.name as service_name, s.price as service_price, s.description
	FROM service s
	INNER JOIN users u ON s.user_id = u.id
	INNER JOIN categories c ON s.category_id = c.id
	`

	// Category filter
	if len(req.Categories) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(req.Categories)), ",")
		conditions = append(conditions, fmt.Sprintf("s.category_id IN (%s)", placeholders))
		for _, catID := range req.Categories {
			params = append(params, catID)
		}
	}

	// Subcategory filter
	if len(req.Subcategories) > 0 {
		for _, sub := range req.Subcategories {
			conditions = append(conditions, "c.subcategories LIKE ?")
			params = append(params, "%"+sub+"%")
		}
	}

	// Price filter
	if req.PriceFrom > 0 {
		conditions = append(conditions, "s.price >= ?")
		params = append(params, req.PriceFrom)
	}
	if req.PriceTo > 0 {
		conditions = append(conditions, "s.price <= ?")
		params = append(params, req.PriceTo)
	}

	// Rating filter
	if len(req.Ratings) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(req.Ratings)), ",")
		conditions = append(conditions, fmt.Sprintf("u.review_rating IN (%s)", placeholders))
		for _, rate := range req.Ratings {
			params = append(params, rate)
		}
	}

	// Final WHERE clause
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += ` ORDER BY (SELECT COUNT(*) FROM reviews r WHERE r.service_id = s.id) DESC `
	case 2:
		query += ` ORDER BY s.price ASC `
	case 3:
		query += ` ORDER BY s.price DESC `
	default:
		query += ` ORDER BY s.created_at DESC `
	}

	rows, err := r.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.GetServicesPostResponse
		if err := rows.Scan(
			&s.ClientID, &s.ClientName, &s.ClientReviewRating,
			&s.ServiceID, &s.ServiceName, &s.ServicePrice, &s.ServiceDescription,
		); err != nil {
			return nil, err
		}
		services = append(services, s)
	}

	return services, nil
}

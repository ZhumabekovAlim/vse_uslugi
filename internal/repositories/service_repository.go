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

//type ServiceResponse struct {
//	ClientID           int     `json:"client_id"`
//	ClientName         string  `json:"client_name"`
//	ClientRating       float64 `json:"client_rating"`
//	ServiceID          int     `json:"service_id"`
//	ServiceName        string  `json:"service_name"`
//	ServicePrice       float64 `json:"service_price"`
//	ServiceDescription string  `json:"service_description"`
//}
//
//type GetServicesRequest struct {
//	Categories    []int     `json:"categories"`
//	Subcategories []string  `json:"subcategories"`
//	PriceFrom     float64   `json:"price_from"`
//	PriceTo       float64   `json:"price_to"`
//	Ratings       []float64 `json:"ratings"`
//	Sorting       string    `json:"sorting"`
//	Page          int       `json:"page"`
//	PageSize      int       `json:"page_size"`
//}

//func (r *ServiceRepository) GetServicesWithFilters(ctx context.Context, filters models.GetServicesRequest) ([]models.ServiceResponse, float64, float64, int, error) {
//	whereClause, args := buildWhereClause(filters)
//
//	aggQuery := fmt.Sprintf(`
//        SELECT MIN(s.price), MAX(s.price), COUNT(*)
//        FROM services s
//        JOIN categories c ON s.category_id = c.id
//        WHERE %s
//    `, whereClause)
//
//	var minPrice, maxPrice float64
//	var total int
//	row := r.DB.QueryRowContext(ctx, aggQuery, args...)
//	err := row.Scan(&minPrice, &maxPrice, &total)
//	if err != nil {
//		return nil, 0, 0, 0, err
//	}
//
//	query := fmt.Sprintf(`
//        SELECT s.id as service_id, s.name as service_name, s.price as service_price, s.description as service_description,
//               u.id as client_id, u.name as client_name, u.review_rating as client_rating
//        FROM services s
//        JOIN users u ON s.user_id = u.id
//        JOIN categories c ON s.category_id = c.id
//        WHERE %s
//        ORDER BY %s
//        LIMIT ? OFFSET ?
//    `, whereClause, getOrderBy(filters.Sorting))
//
//	offset := (filters.Page - 1) * filters.PageSize
//	args = append(args, filters.PageSize, offset)
//
//	rows, err := r.DB.QueryContext(ctx, query, args...)
//	if err != nil {
//		return nil, 0, 0, 0, err
//	}
//	defer rows.Close()
//
//	var services []models.ServiceResponse
//	for rows.Next() {
//		var srv models.ServiceResponse
//		err := rows.Scan(
//			&srv.ServiceID, &srv.ServiceName, &srv.ServicePrice, &srv.ServiceDescription,
//			&srv.ClientID, &srv.ClientName, &srv.ClientRating,
//		)
//		if err != nil {
//			return nil, 0, 0, 0, err
//		}
//		services = append(services, srv)
//	}
//	return services, minPrice, maxPrice, total, nil
//}

//func buildWhereClause(filters models.GetServicesRequest) (string, []interface{}) {
//	var conditions []string
//	var args []interface{}
//
//	if len(filters.Categories) > 0 {
//		placeholders := strings.Repeat("?,", len(filters.Categories)-1) + "?"
//		conditions = append(conditions, fmt.Sprintf("s.category_id IN (%s)", placeholders))
//		for _, cat := range filters.Categories {
//			args = append(args, cat)
//		}
//	}
//
//	if len(filters.Subcategories) > 0 {
//		for _, subcat := range filters.Subcategories {
//			conditions = append(conditions, "find_in_set(?, c.subcategories) > 0")
//			args = append(args, subcat)
//		}
//	}
//
//	if filters.PriceFrom > 0 {
//		conditions = append(conditions, "s.price >= ?")
//		args = append(args, filters.PriceFrom)
//	}
//
//	if filters.PriceTo > 0 {
//		conditions = append(conditions, "s.price <= ?")
//		args = append(args, filters.PriceTo)
//	}
//
//	if len(filters.Ratings) > 0 {
//		placeholders := strings.Repeat("?,", len(filters.Ratings)-1) + "?"
//		conditions = append(conditions, fmt.Sprintf("s.avg_rating IN (%s)", placeholders))
//		for _, rating := range filters.Ratings {
//			args = append(args, rating)
//		}
//	}
//
//	if len(conditions) == 0 {
//		return "1=1", args
//	}
//	return strings.Join(conditions, " AND "), args
//}
//
//func getOrderBy(sorting string) string {
//	switch sorting {
//	case "price_asc":
//		return "s.price ASC"
//	case "price_desc":
//		return "s.price DESC"
//	case "popularity":
//		return "s.avg_rating DESC"
//	default:
//		return "s.created_at DESC"
//	}
//}

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

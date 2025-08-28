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
	ErrRentAdNotFound = errors.New("service not found")
)

type RentAdRepository struct {
	DB *sql.DB
}

func (r *RentAdRepository) CreateRentAd(ctx context.Context, rent models.RentAd) (models.RentAd, error) {
	query := `
        INSERT INTO rent_ad (name, address, price, user_id, images, category_id, subcategory_id, description, avg_rating, top, liked, status, rent_type, deposit, latitude, longitude, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(rent.Images)
	if err != nil {
		return models.RentAd{}, err
	}

	result, err := r.DB.ExecContext(ctx, query,
		rent.Name,
		rent.Address,
		rent.Price,
		rent.UserID,
		string(imagesJSON),
		rent.CategoryID,
		rent.SubcategoryID,
		rent.Description,
		rent.AvgRating,
		rent.Top,
		rent.Liked,
		rent.Status,
		rent.RentType,
		rent.Deposit,
		rent.Latitude,
		rent.Longitude,
		rent.CreatedAt,
	)
	if err != nil {
		return models.RentAd{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.RentAd{}, err
	}
	rent.ID = int(lastID)
	return rent, nil
}

func (r *RentAdRepository) GetRentAdByID(ctx context.Context, id int, userID int) (models.RentAd, error) {
	query := `
               SELECT w.id, w.name, w.address, w.price, w.user_id, u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path, w.images, w.category_id, c.name, w.subcategory_id, sub.name, w.description, w.avg_rating, w.top, w.liked,
                      CASE WHEN sr.id IS NOT NULL THEN true ELSE false END AS responded,
                      w.status, w.rent_type, w.deposit, w.latitude, w.longitude, w.created_at, w.updated_at
                FROM rent_ad w
                JOIN users u ON w.user_id = u.id
                JOIN rent_categories c ON w.category_id = c.id
                JOIN rent_subcategories sub ON w.subcategory_id = sub.id
               LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = w.id AND sr.user_id = ?
                WHERE w.id = ?
       `

	var s models.RentAd
	var imagesJSON []byte
	err := r.DB.QueryRowContext(ctx, query, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
		&imagesJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Responded, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt,
		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.RentAd{}, errors.New("not found")
	}
	if err != nil {
		return models.RentAd{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.RentAd{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *RentAdRepository) UpdateRentAd(ctx context.Context, work models.RentAd) (models.RentAd, error) {
	query := `
        UPDATE rent_ad
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, category_id = ?, subcategory_id = ?, 
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, rent_type = ?, deposit = ?, latitude = ?, longitude = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.RentAd{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, work.Price, work.UserID, imagesJSON,
		work.CategoryID, work.SubcategoryID, work.Description, work.AvgRating, work.Top, work.Liked, work.Status, work.RentType, work.Deposit, work.Latitude, work.Longitude, work.UpdatedAt, work.ID,
	)
	if err != nil {
		return models.RentAd{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.RentAd{}, err
	}
	if rowsAffected == 0 {
		return models.RentAd{}, ErrServiceNotFound
	}
	return r.GetRentAdByID(ctx, work.ID, 0)
}

func (r *RentAdRepository) DeleteRentAd(ctx context.Context, id int) error {
	query := `DELETE FROM rent_ad WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRentNotFound
	}
	return nil
}

func (r *RentAdRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE rent_ad SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRentAdNotFound
	}
	return nil
}
func (r *RentAdRepository) GetRentsAdWithFilters(ctx context.Context, userID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int) ([]models.RentAd, float64, float64, error) {
	var (
		rents      []models.RentAd
		params     []interface{}
		conditions []string
	)

	baseQuery := `
               SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path, s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, CASE WHEN sf.rent_ad_id IS NOT NULL THEN 'true' ELSE 'false' END AS liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
               FROM rent_ad s
               LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
               JOIN users u ON s.user_id = u.id
               INNER JOIN rent_categories c ON s.category_id = c.id

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
		baseQuery += ` ORDER BY ( SELECT COUNT(*) FROM rent_ad_reviews r WHERE r.rent_ad_id = s.id) DESC `

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
		var s models.RentAd
		var imagesJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}
		rents = append(rents, s)
	}

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM work`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return rents, 0, 0, nil // fallback
	}

	return rents, minPrice, maxPrice, nil
}

func (r *RentAdRepository) GetRentsAdByUserID(ctx context.Context, userID int) ([]models.RentAd, error) {
	query := `
                SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, u.avatar_path, s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, s.liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
		FROM rent_ad s
		JOIN users u ON s.user_id = u.id
		WHERE user_id = ?
	`

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rents []models.RentAd
	for rows.Next() {
		var s models.RentAd
		var imagesJSON []byte
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(imagesJSON) > 0 {
			if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

		rents = append(rents, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rents, nil
}

func (r *RentAdRepository) GetFilteredRentsAdPost(ctx context.Context, req models.FilterRentAdRequest) ([]models.FilteredRentAd, error) {
	query := `
      SELECT

              u.id, u.name, u.surname, u.phone, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

              s.id, s.name, s.price, s.description, s.latitude, s.longitude
      FROM rent_ad s
      JOIN users u ON s.user_id = u.id
      WHERE 1=1
`
	args := []interface{}{}

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
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
		query += " ORDER BY (SELECT COUNT(*) FROM rent_ad_reviews r WHERE r.rent_ad_id = s.id) DESC"
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

	var rents []models.FilteredRentAd
	for rows.Next() {
		var s models.FilteredRentAd
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserPhone, &s.UserAvatarPath, &s.UserRating,
			&s.RentAdID, &s.RentAdName, &s.RentAdPrice, &s.RentAdDescription, &s.RentAdLatitude, &s.RentAdLongitude,
		); err != nil {
			return nil, err
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		rents = append(rents, s)
	}

	return rents, nil
}

func (r *RentAdRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.RentAd, error) {
	query := `
        SELECT
                s.id, s.name, s.address, s.price, s.user_id,
                u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                s.images, s.category_id, s.subcategory_id, s.description,
                s.avg_rating, s.top, s.liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude,
                s.created_at, s.updated_at
	FROM rent_ad s
	JOIN users u ON s.user_id = u.id
	WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rents []models.RentAd
	for rows.Next() {
		var s models.RentAd
		var imagesJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("json decode error: %w", err)
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)
		rents = append(rents, s)
	}
	return rents, nil
}

func (r *RentAdRepository) GetFilteredRentsAdWithLikes(ctx context.Context, req models.FilterRentAdRequest, userID int) ([]models.FilteredRentAd, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
   SELECT DISTINCT

           u.id, u.name, u.surname, u.phone, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

           s.id, s.name, s.price, s.description, s.latitude, s.longitude,
           CASE WHEN sf.id IS NOT NULL THEN true ELSE false END AS liked,
           CASE WHEN sr.id IS NOT NULL THEN true ELSE false END AS responded
   FROM rent_ad s
   JOIN users u ON s.user_id = u.id
   LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
   LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = s.id AND sr.user_id = ?
   WHERE 1=1
`

	args := []interface{}{userID, userID}

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
	}

	// Category
	if len(req.CategoryIDs) > 0 {
		log.Printf("[DEBUG] Filtering by CategoryIDs: %v", req.CategoryIDs)
		placeholders := strings.Repeat("?,", len(req.CategoryIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.category_id IN (%s)", placeholders)
		for _, id := range req.CategoryIDs {
			args = append(args, id)
		}
	}

	// Subcategory
	if len(req.SubcategoryIDs) > 0 {
		log.Printf("[DEBUG] Filtering by SubcategoryIDs: %v", req.SubcategoryIDs)
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
		log.Printf("[DEBUG] Filtering by minimum AvgRating: %.2f", float64(req.AvgRatings[0]))
		query += " AND s.avg_rating >= ?"
		args = append(args, float64(req.AvgRatings[0]))
	}

	// Sorting
	switch req.Sorting {
	case 1:
		query += " ORDER BY (SELECT COUNT(*) FROM rent_ad_reviews r WHERE r.rent_ad_id = s.id) DESC"
		log.Println("[DEBUG] Sorting by most reviewed")
	case 2:
		query += " ORDER BY s.price DESC"
		log.Println("[DEBUG] Sorting by price DESC")
	case 3:
		query += " ORDER BY s.price ASC"
		log.Println("[DEBUG] Sorting by price ASC")
	default:
		query += " ORDER BY s.created_at DESC"
		log.Println("[DEBUG] Sorting by created_at DESC (default)")
	}

	log.Printf("[DEBUG] Final SQL Query: %s", query)
	log.Printf("[DEBUG] Query Args: %+v", args)

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

	var rents []models.FilteredRentAd
	for rows.Next() {
		var s models.FilteredRentAd
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserPhone, &s.UserAvatarPath, &s.UserRating,
			&s.RentAdID, &s.RentAdName, &s.RentAdPrice, &s.RentAdDescription, &s.RentAdLatitude, &s.RentAdLongitude, &s.Liked, &s.Responded,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		rents = append(rents, s)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error after reading rows: %v", err)
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	log.Printf("[INFO] Successfully fetched %d services", len(rents))
	return rents, nil
}

func (r *RentAdRepository) GetRentAdByRentIDAndUserID(ctx context.Context, rentAdID int, userID int) (models.RentAd, error) {
	query := `
               SELECT
                       s.id, s.name, s.address, s.price, s.user_id,
                       u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                       s.images, s.category_id, c.name,
                       s.subcategory_id, sub.name,
                       s.description, s.avg_rating, s.top,
                       CASE WHEN sf.id IS NOT NULL THEN true ELSE false END AS liked,
                       CASE WHEN sr.id IS NOT NULL THEN true ELSE false END AS responded,
                       s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
               FROM rent_ad s
               JOIN users u ON s.user_id = u.id
               JOIN rent_categories c ON s.category_id = c.id
               JOIN rent_subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
               LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = s.id AND sr.user_id = ?
               WHERE s.id = ?
       `

	var s models.RentAd
	var imagesJSON []byte

	err := r.DB.QueryRowContext(ctx, query, userID, userID, rentAdID).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
		&imagesJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName,
		&s.Description, &s.AvgRating, &s.Top,
		&s.Liked, &s.Responded, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.RentAd{}, errors.New("service not found")
	}
	if err != nil {
		return models.RentAd{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.RentAd{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

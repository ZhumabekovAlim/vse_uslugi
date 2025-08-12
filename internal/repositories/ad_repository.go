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
	ErrAdNotFound = errors.New("ad not found")
)

type AdRepository struct {
	DB *sql.DB
}

func (r *AdRepository) CreateAd(ctx context.Context, ad models.Ad) (models.Ad, error) {
	query := `
        INSERT INTO ad (name, address, price, user_id, images, category_id, subcategory_id, description, avg_rating, top, liked, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(ad.Images)
	if err != nil {
		return models.Ad{}, err
	}

	result, err := r.DB.ExecContext(ctx, query,
		ad.Name,
		ad.Address,
		ad.Price,
		ad.UserID,
		string(imagesJSON),
		ad.CategoryID,
		ad.SubcategoryID,
		ad.Description,
		ad.AvgRating,
		ad.Top,
		ad.Liked,
		ad.Status,
		ad.CreatedAt,
	)
	if err != nil {
		return models.Ad{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.Ad{}, err
	}
	ad.ID = int(lastID)
	return ad, nil
}

func (r *AdRepository) GetAdByID(ctx context.Context, id int) (models.Ad, error) {
	query := `
               SELECT s.id, s.name, s.address, s.price, s.user_id,
                      u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                      s.images, s.category_id, c.name, s.subcategory_id, sub.name,
                      s.description, s.avg_rating, s.top, s.liked, s.status, s.created_at, s.updated_at
               FROM ad s
               JOIN users u ON s.user_id = u.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               WHERE s.id = ?
       `

	var s models.Ad
	var imagesJSON []byte
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
		&imagesJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Ad{}, errors.New("not found")
	}
	if err != nil {
		return models.Ad{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Ad{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "ad_reviews", "ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *AdRepository) UpdateAd(ctx context.Context, service models.Ad) (models.Ad, error) {
	query := `
        UPDATE ad
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, category_id = ?, subcategory_id = ?, 
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(service.Images)
	if err != nil {
		return models.Ad{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	updatedAt := time.Now()
	service.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		service.Name, service.Address, service.Price, service.UserID, imagesJSON,
		service.CategoryID, service.SubcategoryID, service.Description, service.AvgRating, service.Top, service.Liked, service.Status, service.UpdatedAt, service.ID,
	)
	if err != nil {
		return models.Ad{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Ad{}, err
	}
	if rowsAffected == 0 {
		return models.Ad{}, ErrAdNotFound
	}
	return r.GetAdByID(ctx, service.ID)
}

func (r *AdRepository) DeleteAd(ctx context.Context, id int) error {
	query := `DELETE FROM ad WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrAdNotFound
	}
	return nil
}
func (r *AdRepository) GetAdWithFilters(ctx context.Context, userID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int) ([]models.Ad, float64, float64, error) {
	var (
		ads        []models.Ad
		params     []interface{}
		conditions []string
	)

	baseQuery := `
               SELECT s.id, s.name, s.address, s.price, s.user_id,
                      u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                      s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top,
                      CASE WHEN sf.ad_id IS NOT NULL THEN 'true' ELSE 'false' END AS liked,
                      s.status,  s.created_at, s.updated_at
               FROM ad s
               LEFT JOIN ad_favorites sf ON sf.ad_id = s.id AND sf.user_id = ?
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
		var s models.Ad
		var imagesJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "ad_reviews", "ad_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}

		ads = append(ads, s)
	}

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM ad`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return ads, 0, 0, nil // fallback
	}

	return ads, minPrice, maxPrice, nil
}

func (r *AdRepository) GetAdByUserID(ctx context.Context, userID int) ([]models.Ad, error) {
	query := `
                SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.review_rating, u.avatar_path, s.images, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, s.liked, s.status, s.created_at, s.updated_at
		FROM ad s
		JOIN users u ON s.user_id = u.id
		WHERE user_id = ?
	`

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var s models.Ad
		var imagesJSON []byte
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(imagesJSON) > 0 {
			if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "ad_reviews", "ad_id", s.ID)

		ads = append(ads, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ads, nil
}

func (r *AdRepository) GetFilteredAdPost(ctx context.Context, req models.FilterAdRequest) ([]models.FilteredAd, error) {
	query := `
       SELECT
               u.id, u.name, u.surname, u.phone, COALESCE(u.avatar_path, ''), u.review_rating,
               s.id, s.name, s.price, s.description
       FROM ad s
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
		query += " ORDER BY (SELECT COUNT(*) FROM ad_reviews r WHERE r.ad_id = s.id) DESC"
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

	var ads []models.FilteredAd
	for rows.Next() {
		var s models.FilteredAd
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserPhone, &s.UserAvatarPath, &s.UserRating,
			&s.AdID, &s.AdName, &s.AdPrice, &s.AdDescription,
		); err != nil {
			return nil, err
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		ads = append(ads, s)
	}

	return ads, nil
}

func (r *AdRepository) FetchAdByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Ad, error) {
	query := `
        SELECT
                s.id, s.name, s.address, s.price, s.user_id,
                u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                s.images, s.category_id, s.subcategory_id, s.description,
                s.avg_rating, s.top, s.liked, s.status,
                s.created_at, s.updated_at
	FROM ad s
	JOIN users u ON s.user_id = u.id
	WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []models.Ad
	for rows.Next() {
		var s models.Ad
		var imagesJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("json decode error: %w", err)
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "ad_reviews", "ad_id", s.ID)
		ads = append(ads, s)
	}
	return ads, nil
}

func (r *AdRepository) GetFilteredAdWithLikes(ctx context.Context, req models.FilterAdRequest, userID int) ([]models.FilteredAd, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
       SELECT DISTINCT
               u.id, u.name, u.surname, u.phone, COALESCE(u.avatar_path, ''), u.review_rating,
               s.id, s.name, s.price, s.description,
               CASE WHEN sf.id IS NOT NULL THEN true ELSE false END AS liked
       FROM ad s
       JOIN users u ON s.user_id = u.id
       LEFT JOIN ad_favorites sf ON sf.ad_id = s.id AND sf.user_id = ?
       WHERE s.price BETWEEN ? AND ?
`

	args := []interface{}{userID, req.PriceFrom, req.PriceTo}

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
		query += " ORDER BY (SELECT COUNT(*) FROM ad_reviews r WHERE r.ad_id = s.id) DESC"
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

	var ads []models.FilteredAd
	for rows.Next() {
		var s models.FilteredAd
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserPhone, &s.UserAvatarPath, &s.UserRating,
			&s.AdID, &s.AdName, &s.AdPrice, &s.AdDescription, &s.Liked,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		ads = append(ads, s)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error after reading rows: %v", err)
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	log.Printf("[INFO] Successfully fetched %d services", len(ads))
	return ads, nil
}

func (r *AdRepository) GetAdByAdIDAndUserID(ctx context.Context, adID int, userID int) (models.Ad, error) {
	query := `
               SELECT
                       s.id, s.name, s.address, s.price, s.user_id,
                       u.id, u.name, u.surname, u.phone, u.review_rating, u.avatar_path,
                       s.images, s.category_id, c.name,
                       s.subcategory_id, sub.name,
                       s.description, s.avg_rating, s.top,
                       CASE WHEN sf.id IS NOT NULL THEN true ELSE false END AS liked,
                       s.status, s.created_at, s.updated_at
               FROM ad s
               JOIN users u ON s.user_id = u.id
               JOIN categories c ON s.category_id = c.id
               JOIN subcategories sub ON s.subcategory_id = sub.id
               LEFT JOIN ad_favorites sf ON sf.ad_id = s.id AND sf.user_id = ?
               WHERE s.id = ?
       `

	var s models.Ad
	var imagesJSON []byte

	err := r.DB.QueryRowContext(ctx, query, userID, adID).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.Phone, &s.User.ReviewRating, &s.User.AvatarPath,
		&imagesJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName,
		&s.Description, &s.AvgRating, &s.Top,
		&s.Liked, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Ad{}, errors.New("service not found")
	}
	if err != nil {
		return models.Ad{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Ad{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "ad_reviews", "ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

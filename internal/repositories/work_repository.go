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
	ErrWorkNotFound = errors.New("service not found")
)

type WorkRepository struct {
	DB *sql.DB
}

func (r *WorkRepository) CreateWork(ctx context.Context, work models.Work) (models.Work, error) {
	query := `
        INSERT INTO work (name, address, price, user_id, images, videos, category_id, subcategory_id, description, avg_rating, top, liked, status, work_experience, city_id, schedule, distance_work, payment_period, latitude, longitude, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.Work{}, err
	}

	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.Work{}, err
	}

	var subcategory interface{}
	if work.SubcategoryID != 0 {
		subcategory = work.SubcategoryID
	}

	result, err := r.DB.ExecContext(ctx, query,
		work.Name,
		work.Address,
		work.Price,
		work.UserID,
		string(imagesJSON),
		string(videosJSON),
		work.CategoryID,
		subcategory,
		work.Description,
		work.AvgRating,
		work.Top,
		work.Liked,
		work.Status,
		work.WorkExperience,
		work.CityID,
		work.Schedule,
		work.DistanceWork,
		work.PaymentPeriod,
		work.Latitude,
		work.Longitude,
		work.CreatedAt,
	)
	if err != nil {
		return models.Work{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.Work{}, err
	}
	work.ID = int(lastID)
	return work, nil
}

func (r *WorkRepository) GetWorkByID(ctx context.Context, id int, userID int) (models.Work, error) {
	query := `
          SELECT w.id, w.name, w.address, w.price, w.user_id,
                 u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                    w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, sub.name_kz, w.description, w.avg_rating, w.top, w.liked,
                    CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                    w.status, w.work_experience, u.city_id, city.name, city.type, w.schedule, w.distance_work, w.payment_period, w.latitude, w.longitude, w.created_at, w.updated_at
              FROM work w
              JOIN users u ON w.user_id = u.id
              JOIN work_categories c ON w.category_id = c.id
              JOIN work_subcategories sub ON w.subcategory_id = sub.id
              JOIN cities city ON u.city_id = city.id
              LEFT JOIN work_responses sr ON sr.work_id = w.id AND sr.user_id = ?
              WHERE w.id = ?
       `

	var s models.Work
	var imagesJSON []byte
	var videosJSON []byte
	var respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude, &s.CreatedAt,

		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Work{}, errors.New("not found")
	}
	if err != nil {
		return models.Work{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *WorkRepository) UpdateWork(ctx context.Context, work models.Work) (models.Work, error) {
	query := `
        UPDATE work
        SET name = ?, address = ?, price = ?, user_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, work_experience = ?, city_id = ?, schedule = ?, distance_work = ?, payment_period = ?, latitude = ?, longitude = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.Work{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.Work{}, fmt.Errorf("failed to marshal videos: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, work.Price, work.UserID, imagesJSON, videosJSON,
		work.CategoryID, work.SubcategoryID, work.Description, work.AvgRating, work.Top, work.Liked, work.Status, work.WorkExperience, work.CityID, work.Schedule, work.DistanceWork, work.PaymentPeriod, work.Latitude, work.Longitude, work.UpdatedAt, work.ID,
	)
	if err != nil {
		return models.Work{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Work{}, err
	}
	if rowsAffected == 0 {
		return models.Work{}, ErrServiceNotFound
	}
	return r.GetWorkByID(ctx, work.ID, 0)
}

func (r *WorkRepository) DeleteWork(ctx context.Context, id int) error {
	query := `DELETE FROM work WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWorkNotFound
	}
	return nil
}

func (r *WorkRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE work SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkNotFound
	}
	return nil
}
func (r *WorkRepository) GetWorksWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int) ([]models.Work, float64, float64, error) {
	var (
		works      []models.Work
		params     []interface{}
		conditions []string
	)

	conditions = append(conditions, "s.status = 'active'")

	baseQuery := `
               SELECT s.id, s.name, s.address, s.price, s.user_id,
                      u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                      s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top,

                     CASE WHEN sf.work_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

                     s.status, s.work_experience, u.city_id, city.name, city.type, s.schedule, s.distance_work, s.payment_period, s.latitude, s.longitude, s.created_at, s.updated_at
              FROM work s
              LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
              JOIN users u ON s.user_id = u.id
              INNER JOIN work_categories c ON s.category_id = c.id
              JOIN cities city ON u.city_id = city.id

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
		baseQuery += ` ORDER BY ( SELECT COUNT(*) FROM work_reviews r WHERE r.work_id = s.id) DESC `

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
		var s models.Work
		var imagesJSON []byte
		var videosJSON []byte
		var likedStr string
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &likedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude, &s.CreatedAt,

			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
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

		s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}

		works = append(works, s)
	}

	sortWorksByTop(works)

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM work`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return works, 0, 0, nil // fallback
	}

	return works, minPrice, maxPrice, nil
}

func (r *WorkRepository) GetWorksByUserID(ctx context.Context, userID int) ([]models.Work, error) {
	query := `
             SELECT s.id, s.name, s.address, s.price, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, s.liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.latitude, s.longitude, s.created_at, s.updated_at
                FROM work s
                JOIN users u ON s.user_id = u.id
                WHERE user_id = ?
       `

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var s models.Work
		var imagesJSON []byte
		var videosJSON []byte
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude, &s.CreatedAt,
			&s.UpdatedAt,
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
				return nil, fmt.Errorf("json decode videos error: %w", err)
			}
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)

		works = append(works, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortWorksByTop(works)

	return works, nil
}

func (r *WorkRepository) GetFilteredWorksPost(ctx context.Context, req models.FilterWorkRequest) ([]models.FilteredWork, error) {
	query := `
SELECT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at
FROM work s
JOIN users u ON s.user_id = u.id
WHERE 1=1
`
	args := []interface{}{}

	query += " AND s.status = 'active'"

	if req.CityID > 0 {
		query += " AND u.city_id = ?"
		args = append(args, req.CityID)
	}

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
		query += " ORDER BY (SELECT COUNT(*) FROM work_reviews r WHERE r.work_id = s.id) DESC"
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

	var works []models.FilteredWork
	for rows.Next() {
		var s models.FilteredWork
		var lat, lon sql.NullString
		var imagesJSON, videosJSON []byte
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,
			&s.WorkID, &s.WorkName, &s.WorkAddress, &s.WorkPrice, &s.WorkDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if lat.Valid {
			s.WorkLatitude = lat.String
		}
		if lon.Valid {
			s.WorkLongitude = lon.String
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
		works = append(works, s)
	}

	sortFilteredWorksByTop(works)
	return works, nil
}

func (r *WorkRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Work, error) {
	query := `
        SELECT
                s.id, s.name, s.address, s.price, s.user_id,
                u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                s.images, s.videos, s.category_id, s.subcategory_id, s.description,
                s.avg_rating, s.top, s.liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.latitude, s.longitude,
                s.created_at, s.updated_at
        FROM work s
        JOIN users u ON s.user_id = u.id
        WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var s models.Work
		var imagesJSON []byte
		var videosJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude,
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
				return nil, fmt.Errorf("json decode videos error: %w", err)
			}
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)
		works = append(works, s)
	}
	sortWorksByTop(works)
	return works, nil
}

func (r *WorkRepository) GetFilteredWorksWithLikes(ctx context.Context, req models.FilterWorkRequest, userID int) ([]models.FilteredWork, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
SELECT DISTINCT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at,
CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
FROM work s
JOIN users u ON s.user_id = u.id
LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
LEFT JOIN work_responses sr ON sr.work_id = s.id AND sr.user_id = ?
WHERE 1=1
`

	args := []interface{}{userID, userID}

	query += " AND s.status = 'active'"

	if req.CityID > 0 {
		query += " AND u.city_id = ?"
		args = append(args, req.CityID)
	}

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
		query += " ORDER BY (SELECT COUNT(*) FROM work_reviews r WHERE r.work_id = s.id) DESC"
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

	var works []models.FilteredWork
	for rows.Next() {
		var s models.FilteredWork
		var lat, lon sql.NullString
		var imagesJSON, videosJSON []byte
		var likedStr, respondedStr string
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.WorkID, &s.WorkName, &s.WorkAddress, &s.WorkPrice, &s.WorkDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if lat.Valid {
			s.WorkLatitude = lat.String
		}
		if lon.Valid {
			s.WorkLongitude = lon.String
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
		works = append(works, s)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error after reading rows: %v", err)
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	sortFilteredWorksByTop(works)
	log.Printf("[INFO] Successfully fetched %d services", len(works))
	return works, nil
}

func (r *WorkRepository) GetWorkByWorkIDAndUserID(ctx context.Context, workID int, userID int) (models.Work, error) {
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
              s.status, s.work_experience, u.city_id, city.name, city.type, s.schedule, s.distance_work, s.payment_period, s.latitude, s.longitude, s.created_at, s.updated_at
               FROM work s
               JOIN users u ON s.user_id = u.id
               JOIN work_categories c ON s.category_id = c.id
               JOIN work_subcategories sub ON s.subcategory_id = sub.id
               JOIN cities city ON u.city_id = city.id
               LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
               LEFT JOIN work_responses sr ON sr.work_id = s.id AND sr.user_id = ?
               WHERE s.id = ?
       `

	var s models.Work
	var imagesJSON []byte
	var videosJSON []byte

	var likedStr, respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, userID, workID).Scan(
		&s.ID, &s.Name, &s.Address, &s.Price, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.Description, &s.AvgRating, &s.Top,
		&likedStr, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Work{}, errors.New("service not found")
	}
	if err != nil {
		return models.Work{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

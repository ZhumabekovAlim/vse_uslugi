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
	"strconv"
	"strings"
	"time"
)

var (
	ErrRentNotFound = errors.New("service not found")
)

type RentRepository struct {
	DB *sql.DB
}

func (r *RentRepository) CreateRent(ctx context.Context, rent models.Rent) (models.Rent, error) {
	query := `
    INSERT INTO rent (name, address, price, price_to, user_id, images, videos, category_id, subcategory_id, work_time_from, work_time_to, description, avg_rating, top, negotiable, hide_phone, liked, status, rent_type, deposit, latitude, longitude, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(rent.Images)
	if err != nil {
		return models.Rent{}, err
	}

	videosJSON, err := json.Marshal(rent.Videos)
	if err != nil {
		return models.Rent{}, err
	}

	var subcategory interface{}
	if rent.SubcategoryID != 0 {
		subcategory = rent.SubcategoryID
	}

	var price, priceTo interface{}
	if rent.Price != nil {
		price = *rent.Price
	}
	if rent.PriceTo != nil {
		priceTo = *rent.PriceTo
	}

	result, err := r.DB.ExecContext(ctx, query,
		rent.Name,
		rent.Address,
		price,
		priceTo,
		rent.UserID,
		string(imagesJSON),
		string(videosJSON),
		rent.CategoryID,
		subcategory,
		rent.WorkTimeFrom,
		rent.WorkTimeTo,
		rent.Description,
		rent.AvgRating,
		rent.Top,
		rent.Negotiable,
		rent.HidePhone,
		rent.Liked,
		rent.Status,
		rent.RentType,
		rent.Deposit,
		rent.Latitude,
		rent.Longitude,
		rent.CreatedAt,
	)
	if err != nil {
		return models.Rent{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.Rent{}, err
	}
	rent.ID = int(lastID)
	return rent, nil
}

func (r *RentRepository) GetRentByID(ctx context.Context, id int, userID int) (models.Rent, error) {
	query := `
             SELECT w.id, w.name, w.address, w.price, w.price_to, w.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, sub.name_kz, w.work_time_from, w.work_time_to, w.description, w.avg_rating, w.top, w.negotiable, w.hide_phone, w.liked,

                      CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,

                      w.status, w.rent_type, w.deposit, w.latitude, w.longitude, w.created_at, w.updated_at
                FROM rent w
                JOIN users u ON w.user_id = u.id
                JOIN rent_categories c ON w.category_id = c.id
                JOIN rent_subcategories sub ON w.subcategory_id = sub.id
                LEFT JOIN rent_responses sr ON sr.rent_id = w.id AND sr.user_id = ?
                WHERE w.id = ?
       `

	var s models.Rent
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var price, priceTo sql.NullFloat64
	var respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &respondedStr, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt,

		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Rent{}, errors.New("not found")
	}
	if err != nil {
		return models.Rent{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Rent{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Rent{}, fmt.Errorf("failed to decode videos json: %w", err)
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
		s.Latitude = lat.String
	}
	if lon.Valid {
		s.Longitude = lon.String
	}
	s.AvgRating = getAverageRating(ctx, r.DB, "rent_reviews", "rent_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *RentRepository) UpdateRent(ctx context.Context, work models.Rent) (models.Rent, error) {
	query := `
UPDATE rent
SET name = ?, address = ?, price = ?, price_to = ?, user_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
    work_time_from = ?, work_time_to = ?, description = ?, avg_rating = ?, top = ?, negotiable = ?, hide_phone = ?, liked = ?, status = ?, rent_type = ?, deposit = ?, latitude = ?, longitude = ?, updated_at = ?
WHERE id = ?
`
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.Rent{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.Rent{}, fmt.Errorf("failed to marshal videos: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	var price, priceTo interface{}
	if work.Price != nil {
		price = *work.Price
	}
	if work.PriceTo != nil {
		priceTo = *work.PriceTo
	}
	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, price, priceTo, work.UserID, imagesJSON, videosJSON,
		work.CategoryID, work.SubcategoryID, work.WorkTimeFrom, work.WorkTimeTo, work.Description, work.AvgRating, work.Top, work.Negotiable, work.HidePhone, work.Liked, work.Status, work.RentType, work.Deposit, work.Latitude, work.Longitude, work.UpdatedAt, work.ID,
	)
	if err != nil {
		return models.Rent{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Rent{}, err
	}
	if rowsAffected == 0 {
		return models.Rent{}, ErrServiceNotFound
	}
	return r.GetRentByID(ctx, work.ID, 0)
}

func (r *RentRepository) DeleteRent(ctx context.Context, id int) error {
	query := `DELETE FROM rent WHERE id = ?`
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

func (r *RentRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE rent SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRentNotFound
	}
	return nil
}

func (r *RentRepository) ArchiveByUserID(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE rent SET status = 'archive', updated_at = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}
func (r *RentRepository) GetRentsWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int, negotiable *bool, rentTypes []string, deposits []string) ([]models.Rent, float64, float64, error) {
	var (
		rents      []models.Rent
		params     []interface{}
		conditions []string
	)

	baseQuery := `

       SELECT s.id, s.name, s.address, s.price, s.price_to, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description, s.avg_rating, s.top, s.negotiable, s.hide_phone, CASE WHEN sf.rent_id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at

               FROM rent s
               LEFT JOIN rent_favorites sf ON sf.rent_id = s.id AND sf.user_id = ?
               JOIN users u ON s.user_id = u.id
               INNER JOIN rent_categories c ON s.category_id = c.id

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

	if negotiable != nil {
		conditions = append(conditions, "s.negotiable = ?")
		params = append(params, *negotiable)
	}

	if len(rentTypes) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(rentTypes)), ",")
		conditions = append(conditions, fmt.Sprintf("s.rent_type IN (%s)", placeholders))
		for _, rentType := range rentTypes {
			params = append(params, rentType)
		}
	}

	if len(deposits) > 0 {
		hasZero := false
		hasPositive := false

		for _, ds := range deposits {
			// если у тебя депозит может быть дробным — оставь ParseFloat
			// если только целым — можно Atoi
			d, err := strconv.ParseFloat(ds, 64)
			if err != nil {
				continue // или return ошибку, как тебе надо
			}
			if d <= 0 {
				hasZero = true
			} else {
				hasPositive = true
			}
		}

		switch {
		case hasZero && hasPositive:
			// выбрали и 0, и >0 => фильтр не ставим (показываем всё)
		case hasPositive:
			conditions = append(conditions, "COALESCE(s.deposit, 0) > 0")
		case hasZero:
			conditions = append(conditions, "COALESCE(s.deposit, 0) = 0")
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
		var s models.Rent
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		var likedStr string
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &likedStr, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if lat.Valid {
			s.Latitude = lat.String
		}
		if lon.Valid {
			s.Longitude = lon.String
		}

		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
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

		s.AvgRating = getAverageRating(ctx, r.DB, "rent_reviews", "rent_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}
		rents = append(rents, s)
	}

	sortRentsByTop(rents)

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM rent`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return rents, 0, 0, nil // fallback
	}

	return rents, minPrice, maxPrice, nil
}

func (r *RentRepository) GetRentsByUserID(ctx context.Context, userID int) ([]models.Rent, error) {
	query := `
       SELECT s.id, s.name, s.address, s.price, s.price_to, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description, s.avg_rating, s.top, s.negotiable, s.hide_phone, s.liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
                FROM rent s
                JOIN users u ON s.user_id = u.id
                WHERE user_id = ?
       `

	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rents []models.Rent
	for rows.Next() {
		var s models.Rent
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lat.Valid {
			s.Latitude = lat.String
		}
		if lon.Valid {
			s.Longitude = lon.String
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
				return nil, fmt.Errorf("json decode videos error: %w", err)
			}
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "rent_reviews", "rent_id", s.ID)

		rents = append(rents, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortRentsByTop(rents)

	return rents, nil
}

func (r *RentRepository) GetFilteredRentsPost(ctx context.Context, req models.FilterRentRequest) ([]models.FilteredRent, error) {
	query := `
SELECT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.work_time_from, s.work_time_to, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at
FROM rent s
JOIN users u ON s.user_id = u.id
WHERE 1=1
`
	args := []interface{}{}

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
	}

	if req.Negotiable != nil {
		query += " AND s.negotiable = ?"
		args = append(args, *req.Negotiable)
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

	if len(req.RentTypes) > 0 {
		placeholders := strings.Repeat("?,", len(req.RentTypes))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.rent_type IN (%s)", placeholders)
		for _, rentType := range req.RentTypes {
			args = append(args, rentType)
		}
	}

	if len(req.Deposits) > 0 {
		placeholders := strings.Repeat("?,", len(req.Deposits))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.deposit IN (%s)", placeholders)
		for _, deposit := range req.Deposits {
			args = append(args, deposit)
		}
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
		query += " ORDER BY (SELECT COUNT(*) FROM rent_reviews r WHERE r.rent_id = s.id) DESC"
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

	var rents []models.FilteredRent
	for rows.Next() {
		var s models.FilteredRent
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		var imagesJSON, videosJSON []byte
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,
			&s.RentID, &s.RentName, &s.RentAddress, &price, &priceTo, &s.RentNegotiable, &s.RentHidePhone, &s.WorkTimeFrom, &s.WorkTimeTo, &s.RentDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if price.Valid {
			s.RentPrice = &price.Float64
		}
		if priceTo.Valid {
			s.RentPriceTo = &priceTo.Float64
		}
		var latPtr, lonPtr *string
		if lat.Valid {
			s.RentLatitude = lat.String
			latPtr = &s.RentLatitude
		}
		if lon.Valid {
			s.RentLongitude = lon.String
			lonPtr = &s.RentLongitude
		}
		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, latPtr, lonPtr)
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
		rents = append(rents, s)
	}

	sortFilteredRentsByTop(rents)
	return rents, nil
}

func (r *RentRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Rent, error) {
	query := `
SELECT
        s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id,
        u.id, u.name, u.surname, u.review_rating, u.avatar_path,
        s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description,
        s.avg_rating, s.top, s.liked, s.status, s.rent_type, s.deposit, s.latitude, s.longitude,
        s.created_at, s.updated_at
FROM rent s
JOIN users u ON s.user_id = u.id
WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rents []models.Rent
	for rows.Next() {
		var s models.Rent
		var imagesJSON []byte
		var videosJSON []byte
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if lat.Valid {
			s.Latitude = lat.String
		}
		if lon.Valid {
			s.Longitude = lon.String
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("json decode error: %w", err)
		}
		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, fmt.Errorf("json decode videos error: %w", err)
			}
		}
		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "rent_reviews", "rent_id", s.ID)
		rents = append(rents, s)
	}
	sortRentsByTop(rents)
	return rents, nil
}

func (r *RentRepository) GetFilteredRentsWithLikes(ctx context.Context, req models.FilterRentRequest, userID int) ([]models.FilteredRent, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
SELECT DISTINCT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.work_time_from, s.work_time_to, s.description, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at,
CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
FROM rent s
JOIN users u ON s.user_id = u.id
LEFT JOIN rent_favorites sf ON sf.rent_id = s.id AND sf.user_id = ?
LEFT JOIN rent_responses sr ON sr.rent_id = s.id AND sr.user_id = ?
WHERE 1=1
`

	args := []interface{}{userID, userID}

	// Price filter (optional)
	if req.PriceFrom > 0 && req.PriceTo > 0 {
		query += " AND s.price BETWEEN ? AND ?"
		args = append(args, req.PriceFrom, req.PriceTo)
	}

	if req.Negotiable != nil {
		query += " AND s.negotiable = ?"
		args = append(args, *req.Negotiable)
	}

	if req.CityID > 0 {
		query += " AND u.city_id = ?"
		args = append(args, req.CityID)
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

	if len(req.RentTypes) > 0 {
		placeholders := strings.Repeat("?,", len(req.RentTypes))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.rent_type IN (%s)", placeholders)
		for _, rentType := range req.RentTypes {
			args = append(args, rentType)
		}
	}

	if len(req.Deposits) > 0 {
		placeholders := strings.Repeat("?,", len(req.Deposits))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.deposit IN (%s)", placeholders)
		for _, deposit := range req.Deposits {
			args = append(args, deposit)
		}
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
		query += " ORDER BY (SELECT COUNT(*) FROM rent_reviews r WHERE r.rent_id = s.id) DESC"
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

	var rents []models.FilteredRent
	for rows.Next() {
		var s models.FilteredRent
		var lat, lon sql.NullString
		var price, priceTo sql.NullFloat64
		var imagesJSON, videosJSON []byte
		var likedStr, respondedStr string
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.RentID, &s.RentName, &s.RentAddress, &price, &priceTo, &s.RentNegotiable, &s.RentHidePhone, &s.WorkTimeFrom, &s.WorkTimeTo, &s.RentDescription, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if price.Valid {
			s.RentPrice = &price.Float64
		}
		if priceTo.Valid {
			s.RentPriceTo = &priceTo.Float64
		}
		var latPtr, lonPtr *string
		if lat.Valid {
			s.RentLatitude = lat.String
			latPtr = &s.RentLatitude
		}
		if lon.Valid {
			s.RentLongitude = lon.String
			lonPtr = &s.RentLongitude
		}
		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, latPtr, lonPtr)
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
		rents = append(rents, s)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error after reading rows: %v", err)
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	sortFilteredRentsByTop(rents)
	log.Printf("[INFO] Successfully fetched %d services", len(rents))
	return rents, nil
}

func (r *RentRepository) GetRentByRentIDAndUserID(ctx context.Context, rentID int, userID int) (models.Rent, error) {
	query := `
    SELECT
    s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id,
    u.id, u.name, u.surname, u.review_rating, u.avatar_path,
      CASE WHEN s.hide_phone = 0 AND sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
    s.images, s.videos, s.category_id, c.name,
       s.subcategory_id, sub.name, sub.name_kz, s.work_time_from, s.work_time_to,
       s.description, s.avg_rating, s.top,
       CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
       CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
       s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
       FROM rent s
       JOIN users u ON s.user_id = u.id
       JOIN rent_categories c ON s.category_id = c.id
       JOIN rent_subcategories sub ON s.subcategory_id = sub.id
       LEFT JOIN rent_favorites sf ON sf.rent_id = s.id AND sf.user_id = ?
       LEFT JOIN rent_responses sr ON sr.rent_id = s.id AND sr.user_id = ?
       WHERE s.id = ?
`

	var s models.Rent
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var price, priceTo sql.NullFloat64

	var likedStr, respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, userID, rentID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz, &s.WorkTimeFrom, &s.WorkTimeTo,
		&s.Description, &s.AvgRating, &s.Top,
		&likedStr, &respondedStr, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Rent{}, errors.New("service not found")
	}
	if err != nil {
		return models.Rent{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.Rent{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.Rent{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	if lat.Valid {
		s.Latitude = lat.String
	}
	if lon.Valid {
		s.Longitude = lon.String
	}
	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "rent_reviews", "rent_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

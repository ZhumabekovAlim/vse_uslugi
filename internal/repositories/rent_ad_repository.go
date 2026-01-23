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
	ErrRentAdNotFound = errors.New("service not found")
)

type RentAdRepository struct {
	DB *sql.DB
}

func (r *RentAdRepository) CreateRentAd(ctx context.Context, rent models.RentAd) (models.RentAd, error) {
	query := `
INSERT INTO rent_ad (
  name, address, price, price_to, user_id, city_id, images, videos,
  category_id, subcategory_id,
  work_time_from, work_time_to,
  description, ` + "`condition`" + `, delivery,
  avg_rating, top, negotiable, hide_phone, liked, ` + "`status`" + `,
  rent_type, deposit, latitude, longitude,
  order_date, order_time, created_at
)
VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
  ?, ?,
  ?, ?, ?,
  ?, ?, ?, ?, ?, ?,
  ?, ?, ?, ?,
  ?, ?, ?
);
`

	// Сохраняем images как JSON
	imagesJSON, err := json.Marshal(rent.Images)
	if err != nil {
		return models.RentAd{}, err
	}

	videosJSON, err := json.Marshal(rent.Videos)
	if err != nil {
		return models.RentAd{}, err
	}

	var subcategory interface{}
	if rent.SubcategoryID != 0 {
		subcategory = rent.SubcategoryID
	}

	var price interface{}
	if rent.Price != nil {
		price = *rent.Price
	}

	var priceTo interface{}
	if rent.PriceTo != nil {
		priceTo = *rent.PriceTo
	}

	var orderDate interface{}
	if rent.OrderDate != nil && *rent.OrderDate != "" {
		orderDate = *rent.OrderDate
	}

	var orderTime interface{}
	if rent.OrderTime != nil && *rent.OrderTime != "" {
		orderTime = *rent.OrderTime
	}

	var cityID interface{}
	if rent.CityID != 0 {
		cityID = rent.CityID
	}

	result, err := r.DB.ExecContext(ctx, query,
		rent.Name,
		rent.Address,
		price,
		priceTo,
		rent.UserID,
		cityID,
		string(imagesJSON),
		string(videosJSON),
		rent.CategoryID,
		subcategory,
		rent.WorkTimeFrom,
		rent.WorkTimeTo,
		rent.Description,
		rent.Condition,
		rent.Delivery,
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
		orderDate,
		orderTime,
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
     SELECT w.id, w.name, w.address, w.price, w.price_to, w.user_id, w.city_id, city.name, u.id, u.name, u.surname, u.review_rating, u.avatar_path, w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, w.work_time_from, w.work_time_to, w.description, w.condition, w.delivery, w.avg_rating, w.top, w.negotiable, w.hide_phone,
            CASE WHEN sf.rent_ad_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

              CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,

              w.order_date, w.order_time, w.status, w.rent_type, w.deposit, w.latitude, w.longitude, w.created_at, w.updated_at
       FROM rent_ad w
       JOIN users u ON w.user_id = u.id
       LEFT JOIN cities city ON w.city_id = city.id
       JOIN rent_categories c ON w.category_id = c.id
       JOIN rent_subcategories sub ON w.subcategory_id = sub.id
      LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = w.id AND sr.user_id = ?
      LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = w.id AND sf.user_id = ?
       WHERE w.id = ? AND w.status <> 'archive'
`

	var s models.RentAd
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var orderDate, orderTime sql.NullString
	var price, priceTo sql.NullFloat64
	var respondedStr string
	var condition sql.NullString
	var delivery sql.NullBool

	err := r.DB.QueryRowContext(ctx, query, userID, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &respondedStr, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt,

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

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.RentAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Responded = respondedStr == "1"

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
	if condition.Valid {
		s.Condition = condition.String
	}
	if delivery.Valid {
		s.Delivery = delivery.Bool
	}

	if lat.Valid {
		s.Latitude = lat.String
	}
	if lon.Valid {
		s.Longitude = lon.String
	}
	if orderDate.Valid {
		s.OrderDate = &orderDate.String
	}
	if orderTime.Valid {
		s.OrderTime = &orderTime.String
	}
	s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *RentAdRepository) GetRentAdByIDWithCity(ctx context.Context, id int, userID int, cityID int) (models.RentAd, error) {
	query := `
     SELECT w.id, w.name, w.address, w.price, w.price_to, w.user_id, w.city_id, city.name, u.id, u.name, u.surname, u.review_rating, u.avatar_path, w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, w.work_time_from, w.work_time_to, w.description, w.condition, w.delivery, w.avg_rating, w.top, w.negotiable, w.hide_phone, w.liked,

              CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,

              w.order_date, w.order_time, w.status, w.rent_type, w.deposit, w.latitude, w.longitude, w.created_at, w.updated_at
       FROM rent_ad w
       JOIN users u ON w.user_id = u.id
       LEFT JOIN cities city ON w.city_id = city.id
       JOIN rent_categories c ON w.category_id = c.id
       JOIN rent_subcategories sub ON w.subcategory_id = sub.id
      LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = w.id AND sr.user_id = ?
       WHERE w.id = ? AND w.status <> 'archive' AND w.city_id = ?
`

	var s models.RentAd
	var imagesJSON []byte
	var videosJSON []byte
	var lat, lon sql.NullString
	var orderDate, orderTime sql.NullString
	var price, priceTo sql.NullFloat64
	var respondedStr string
	var condition sql.NullString
	var delivery sql.NullBool

	err := r.DB.QueryRowContext(ctx, query, userID, id, cityID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &respondedStr, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &lat, &lon, &s.CreatedAt,

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

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.RentAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Responded = respondedStr == "1"

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
	if condition.Valid {
		s.Condition = condition.String
	}
	if delivery.Valid {
		s.Delivery = delivery.Bool
	}

	if lat.Valid {
		s.Latitude = lat.String
	}
	if lon.Valid {
		s.Longitude = lon.String
	}

	if orderDate.Valid {
		s.OrderDate = &orderDate.String
	}
	if orderTime.Valid {
		s.OrderTime = &orderTime.String
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
    SET name = ?, address = ?, price = ?, price_to = ?, negotiable = ?, hide_phone = ?, user_id = ?, city_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
        work_time_from = ?, work_time_to = ?, description = ?, `condition` = ?, delivery = ?, avg_rating = ?, `top` = ?, liked = ?, status = ?, rent_type = ?, deposit = ?, latitude = ?, longitude = ?, order_date = ?, order_time = ?, updated_at = ?
    WHERE id = ?
`
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.RentAd{}, fmt.Errorf("failed to marshal images: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.RentAd{}, fmt.Errorf("failed to marshal videos: %w", err)
	}
	var price interface{}
	if work.Price != nil {
		price = *work.Price
	}

	var priceTo interface{}
	if work.PriceTo != nil {
		priceTo = *work.PriceTo
	}

	var orderDate interface{}
	if work.OrderDate != nil && *work.OrderDate != "" {
		orderDate = *work.OrderDate
	}

	var orderTime interface{}
	if work.OrderTime != nil && *work.OrderTime != "" {
		orderTime = *work.OrderTime
	}

	var cityID interface{}
	if work.CityID != 0 {
		cityID = work.CityID
	}

	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, price, priceTo, work.Negotiable, work.HidePhone, work.UserID, cityID, imagesJSON, videosJSON,
		work.CategoryID, work.SubcategoryID, work.WorkTimeFrom, work.WorkTimeTo, work.Description, work.Condition, work.Delivery, work.AvgRating, work.Top, work.Liked, work.Status, work.RentType, work.Deposit, work.Latitude, work.Longitude, orderDate, orderTime, work.UpdatedAt, work.ID,
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

func (r *RentAdRepository) GetStatus(ctx context.Context, id int) (string, error) {
	var status string
	if err := r.DB.QueryRowContext(ctx, `SELECT status FROM rent_ad WHERE id = ?`, id).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return "", ErrRentAdNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *RentAdRepository) ArchiveByUserID(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE rent_ad SET status = 'archive', updated_at = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}
func (r *RentAdRepository) GetRentsAdWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int, negotiable *bool, rentTypes []string, deposits []string, condition *string, delivery *bool, orderDate, orderTime *string) ([]models.RentAd, float64, float64, error) {
	var (
		rents      []models.RentAd
		params     []interface{}
		conditions []string
	)

	baseQuery := `

              SELECT s.id, s.name, s.address, s.price, s.price_to, s.user_id, city.name, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery, s.avg_rating, s.top, s.negotiable, s.hide_phone, CASE WHEN sf.rent_ad_id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.order_date, s.order_time, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at

               FROM rent_ad s
               LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
               JOIN users u ON s.user_id = u.id
               LEFT JOIN cities city ON s.city_id = city.id
               INNER JOIN rent_categories c ON s.category_id = c.id

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
			d, err := strconv.ParseFloat(ds, 64)
			if err != nil {
				continue
			}
			if d <= 0 {
				hasZero = true
			} else {
				hasPositive = true
			}
		}

		switch {
		case hasZero && hasPositive:
		case hasPositive:
			conditions = append(conditions, "COALESCE(s.deposit, 0) > 0")
		case hasZero:
			conditions = append(conditions, "COALESCE(s.deposit, 0) = 0")
		}
	}

	if condition != nil && strings.TrimSpace(*condition) != "" {
		conditions = append(conditions, "s.condition = ?")
		params = append(params, strings.TrimSpace(*condition))
	}

	if delivery != nil {
		conditions = append(conditions, "s.delivery = ?")
		params = append(params, *delivery)
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

	if orderDate != nil {
		conditions = append(conditions, "s.order_date = ?")
		params = append(params, *orderDate)
	}

	if orderTime != nil {
		conditions = append(conditions, "s.order_time = ?")
		params = append(params, *orderTime)
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
		var s models.RentAd
		var imagesJSON []byte
		var videosJSON []byte
		var likedStr string
		var orderDate, orderTime sql.NullString
		var price, priceTo sql.NullFloat64
		var condition sql.NullString
		var delivery sql.NullBool
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.CityName, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &likedStr, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		if len(imagesJSON) > 0 {
			if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
			}
		}

		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode videos error: %w", err)
			}
		}

		s.Liked = likedStr == "1"

		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}
		if condition.Valid {
			s.Condition = condition.String
		}
		if delivery.Valid {
			s.Delivery = delivery.Bool
		}

		if orderDate.Valid {
			s.OrderDate = &orderDate.String
		}
		if orderTime.Valid {
			s.OrderTime = &orderTime.String
		}

		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}
		rents = append(rents, s)
	}

	liftListingsTopOnly(rents, func(r models.RentAd) string { return r.Top })

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
                SELECT s.id, s.name, s.address, s.price, s.price_to, s.user_id, s.city_id, city.name, u.id, u.name, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery, s.avg_rating, s.top, s.negotiable, s.hide_phone, CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.order_date, s.order_time, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
                FROM rent_ad s
                JOIN users u ON s.user_id = u.id
                LEFT JOIN cities city ON s.city_id = city.id
                LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
                WHERE s.user_id = ?
       `

	rows, err := r.DB.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rents []models.RentAd
	for rows.Next() {
		var s models.RentAd
		var imagesJSON []byte
		var videosJSON []byte
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var condition sql.NullString
		var delivery sql.NullBool
		var likedStr string
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.CityID, &s.CityName, &s.User.ID, &s.User.Name, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &likedStr, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
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
		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}
		if condition.Valid {
			s.Condition = condition.String
		}
		if delivery.Valid {
			s.Delivery = delivery.Bool
		}
		if orderDate.Valid {
			s.OrderDate = &orderDate.String
		}
		if orderTime.Valid {
			s.OrderTime = &orderTime.String
		}

		s.Liked = likedStr == "1"
		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

		rents = append(rents, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortRentAdsByTop(rents)

	return rents, nil
}

func (r *RentAdRepository) GetFilteredRentsAdPost(ctx context.Context, req models.FilterRentAdRequest) ([]models.FilteredRentAd, error) {
	query := `
      SELECT

              u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

             s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery, s.latitude, s.longitude, s.order_date, s.order_time,
             COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
             s.top, s.created_at
      FROM rent_ad s
      JOIN users u ON s.user_id = u.id
      WHERE 1=1 AND s.status != 'archive'
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

	var rents []models.FilteredRentAd
	for rows.Next() {
		var s models.FilteredRentAd
		var imagesJSON, videosJSON []byte
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var condition sql.NullString
		var delivery sql.NullBool
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,
			&s.RentAdID, &s.RentAdName, &s.RentAdAddress, &price, &priceTo, &s.RentAdNegotiable, &s.RentAdHidePhone, &s.WorkTimeFrom, &s.WorkTimeTo, &s.RentAdDescription, &condition, &delivery, &s.RentAdLatitude, &s.RentAdLongitude, &orderDate, &orderTime,
			&imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}
		if price.Valid {
			s.RentAdPrice = &price.Float64
		}
		if priceTo.Valid {
			s.RentAdPriceTo = &priceTo.Float64
		}
		if condition.Valid {
			s.RentAdCondition = condition.String
		}
		if delivery.Valid {
			s.RentAdDelivery = delivery.Bool
		}
		if orderDate.Valid {
			val := orderDate.String
			s.OrderDate = &val
		}
		if orderTime.Valid {
			val := orderTime.String
			s.OrderTime = &val
		}

		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, &s.RentAdLatitude, &s.RentAdLongitude)
		if req.RadiusKm != nil && (s.Distance == nil || *s.Distance > *req.RadiusKm) {
			continue
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		rents = append(rents, s)
	}

	liftListingsTopOnly(rents, func(r models.FilteredRentAd) string { return r.Top })
	return rents, nil
}

func (r *RentAdRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.RentAd, error) {
	query := `
        SELECT
                s.id, s.name, s.address, s.price, s.price_to, s.user_id,
                u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                s.images, s.videos, s.category_id, s.subcategory_id, s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery,
                s.avg_rating, s.top, s.negotiable, s.hide_phone, s.liked, s.order_date, s.order_time, s.status, s.rent_type, s.deposit, s.latitude, s.longitude,
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
		var videosJSON []byte
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var condition sql.NullString
		var delivery sql.NullBool
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.WorkTimeFrom, &s.WorkTimeTo,
			&s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone, &s.Liked, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
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
		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}
		if condition.Valid {
			s.Condition = condition.String
		}
		if delivery.Valid {
			s.Delivery = delivery.Bool
		}
		if orderDate.Valid {
			s.OrderDate = &orderDate.String
		}
		if orderTime.Valid {
			s.OrderTime = &orderTime.String
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)
		rents = append(rents, s)
	}
	sortRentAdsByTop(rents)
	return rents, nil
}

func (r *RentAdRepository) GetFilteredRentsAdWithLikes(ctx context.Context, req models.FilterRentAdRequest, userID int) ([]models.FilteredRentAd, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
    SELECT DISTINCT

           u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

           s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery, s.latitude, s.longitude, s.order_date, s.order_time,
       COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
       s.top, s.created_at,
           CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
           CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
   FROM rent_ad s
   JOIN users u ON s.user_id = u.id
   LEFT JOIN rent_ad_favorites sf ON sf.rent_ad_id = s.id AND sf.user_id = ?
   LEFT JOIN rent_ad_responses sr ON sr.rent_ad_id = s.id AND sr.user_id = ?
   WHERE 1=1 AND s.status != 'archive'
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
		query += " ORDER BY s.created_at ASC"
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

	var rents []models.FilteredRentAd
	for rows.Next() {
		var s models.FilteredRentAd
		var imagesJSON, videosJSON []byte
		var likedStr, respondedStr string
		var price, priceTo sql.NullFloat64
		var orderDate, orderTime sql.NullString
		var condition sql.NullString
		var delivery sql.NullBool
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.RentAdID, &s.RentAdName, &s.RentAdAddress, &price, &priceTo, &s.RentAdNegotiable, &s.RentAdHidePhone, &s.WorkTimeFrom, &s.WorkTimeTo, &s.RentAdDescription, &condition, &delivery, &s.RentAdLatitude, &s.RentAdLongitude, &orderDate, &orderTime, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}

		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, &s.RentAdLatitude, &s.RentAdLongitude)
		if req.RadiusKm != nil && (s.Distance == nil || *s.Distance > *req.RadiusKm) {
			continue
		}
		s.Liked = likedStr == "1"
		s.Responded = respondedStr == "1"
		if price.Valid {
			s.RentAdPrice = &price.Float64
		}
		if priceTo.Valid {
			s.RentAdPriceTo = &priceTo.Float64
		}
		if condition.Valid {
			s.RentAdCondition = condition.String
		}
		if delivery.Valid {
			s.RentAdDelivery = delivery.Bool
		}
		if orderDate.Valid {
			val := orderDate.String
			s.OrderDate = &val
		}
		if orderTime.Valid {
			val := orderTime.String
			s.OrderTime = &val
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

	liftListingsTopOnly(rents, func(r models.FilteredRentAd) string { return r.Top })
	log.Printf("[INFO] Successfully fetched %d services", len(rents))
	return rents, nil
}

func (r *RentAdRepository) GetRentAdByRentIDAndUserID(ctx context.Context, rentAdID int, userID int) (models.RentAd, error) {
	query := `
            SELECT
                    s.id, s.name, s.address, s.price, s.price_to, s.user_id, s.city_id,
                    u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                      CASE WHEN s.hide_phone = 0 AND sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                       s.images, s.videos, s.category_id, c.name,
                       s.subcategory_id, sub.name,
                       s.work_time_from, s.work_time_to, s.description, s.condition, s.delivery, s.avg_rating, s.top, s.negotiable, s.hide_phone,
                       CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
                       CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
                       s.order_date, s.order_time, s.status, s.rent_type, s.deposit, s.latitude, s.longitude, s.created_at, s.updated_at
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
	var videosJSON []byte
	var price, priceTo sql.NullFloat64
	var orderDate, orderTime sql.NullString
	var condition sql.NullString
	var delivery sql.NullBool

	var likedStr, respondedStr string

	err := r.DB.QueryRowContext(ctx, query, userID, userID, rentAdID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.UserID, &s.CityID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName,
		&s.WorkTimeFrom, &s.WorkTimeTo, &s.Description, &condition, &delivery, &s.AvgRating, &s.Top, &s.Negotiable, &s.HidePhone,
		&likedStr, &respondedStr, &orderDate, &orderTime, &s.Status, &s.RentType, &s.Deposit, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt,
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

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.RentAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}
	if condition.Valid {
		s.Condition = condition.String
	}
	if delivery.Valid {
		s.Delivery = delivery.Bool
	}
	if orderDate.Valid {
		s.OrderDate = &orderDate.String
	}
	if orderTime.Valid {
		s.OrderTime = &orderTime.String
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "rent_ad_reviews", "rent_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

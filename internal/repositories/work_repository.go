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
INSERT INTO work (name, address, price, price_to, negotiable, user_id, images, videos, category_id, subcategory_id, description, avg_rating, top, liked, status, work_experience, city_id, schedule, distance_work, payment_period, languages, education, work_time_from, work_time_to, latitude, longitude, hide_phone, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.Work{}, err
	}

	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.Work{}, err
	}

	languagesJSON, err := json.Marshal(work.Languages)
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
		nullableFloat(work.Price),
		nullableFloat(work.PriceTo),
		work.Negotiable,
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
		string(languagesJSON),
		work.Education,
		work.WorkTimeFrom,
		work.WorkTimeTo,
		work.Latitude,
		work.Longitude,
		work.HidePhone,
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
  SELECT w.id, w.name, w.address, w.price, w.price_to, w.negotiable, w.user_id,
         u.id, u.name, u.surname, u.review_rating, u.avatar_path,
            w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, sub.name_kz, w.description, w.avg_rating, w.top, w.liked,
            CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
            w.status, w.work_experience, u.city_id, city.name, city.type, w.schedule, w.distance_work, w.payment_period, w.languages, w.education, w.work_time_from, w.work_time_to, w.latitude, w.longitude, w.hide_phone, w.created_at, w.updated_at
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
	var languagesJSON []byte
	var respondedStr string
	var price, priceTo sql.NullFloat64
	var negotiable bool
	var hidePhone bool

	err := r.DB.QueryRowContext(ctx, query, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &negotiable, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &hidePhone, &s.CreatedAt,

		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Work{}, errors.New("not found")
	}
	if err != nil {
		return models.Work{}, err
	}

	s.Price = floatFromNull(price)
	s.PriceTo = floatFromNull(priceTo)
	s.Negotiable = negotiable
	s.HidePhone = hidePhone

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

	if len(languagesJSON) > 0 {
		if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode languages json: %w", err)
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
SET name = ?, address = ?, price = ?, price_to = ?, negotiable = ?, user_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
    description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, work_experience = ?, city_id = ?, schedule = ?, distance_work = ?, payment_period = ?, languages = ?, education = ?, work_time_from = ?, work_time_to = ?, latitude = ?, longitude = ?, hide_phone = ?, updated_at = ?
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
	languagesJSON, err := json.Marshal(work.Languages)
	if err != nil {
		return models.Work{}, fmt.Errorf("failed to marshal languages: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, nullableFloat(work.Price), nullableFloat(work.PriceTo), work.Negotiable, work.UserID, imagesJSON, videosJSON,
		work.CategoryID, work.SubcategoryID, work.Description, work.AvgRating, work.Top, work.Liked, work.Status, work.WorkExperience, work.CityID, work.Schedule, work.DistanceWork, work.PaymentPeriod, string(languagesJSON), work.Education, work.WorkTimeFrom, work.WorkTimeTo, work.Latitude, work.Longitude, work.HidePhone, work.UpdatedAt, work.ID,
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

func (r *WorkRepository) ArchiveByUserID(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE work SET status = 'archive', updated_at = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}
func (r *WorkRepository) GetWorksWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int, negotiable *bool, experience []string, schedules []string, paymentPeriods []string, remoteWork *bool, languages []string, educations []string) ([]models.Work, float64, float64, error) {
	var (
		works      []models.Work
		params     []interface{}
		conditions []string
	)

	baseQuery := `
      SELECT s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.user_id,
             u.id, u.name, u.surname, u.review_rating, u.avatar_path,
             s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top,

             CASE WHEN sf.work_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

             s.status, s.work_experience, u.city_id, city.name, city.type, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.hide_phone, s.created_at, s.updated_at
      FROM work s
      LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
      JOIN users u ON s.user_id = u.id
      INNER JOIN work_categories c ON s.category_id = c.id
      JOIN cities city ON u.city_id = city.id

`
	params = append(params, userID)

	conditions = append(conditions, "s.status != 'archive'")

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

	if len(experience) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(experience)), ",")
		conditions = append(conditions, fmt.Sprintf("s.work_experience IN (%s)", placeholders))
		for _, value := range experience {
			params = append(params, value)
		}
	}

	if len(schedules) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(schedules)), ",")
		conditions = append(conditions, fmt.Sprintf("s.schedule IN (%s)", placeholders))
		for _, value := range schedules {
			params = append(params, value)
		}
	}

	if len(paymentPeriods) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(paymentPeriods)), ",")
		conditions = append(conditions, fmt.Sprintf("s.payment_period IN (%s)", placeholders))
		for _, value := range paymentPeriods {
			params = append(params, value)
		}
	}

	if remoteWork != nil {
		if *remoteWork {
			conditions = append(conditions, "s.distance_work = ?")
			params = append(params, "Удаленно")
		} else {
			conditions = append(conditions, "(s.distance_work IS NULL OR s.distance_work <> ?)")
			params = append(params, "Удаленно")
		}
	}

	if len(languages) > 0 {
		for _, lang := range languages {
			conditions = append(conditions, "s.languages LIKE ?")
			params = append(params, "%"+lang+"%")
		}
	}

	if len(educations) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(educations)), ",")
		conditions = append(conditions, fmt.Sprintf("s.education IN (%s)", placeholders))
		for _, value := range educations {
			params = append(params, value)
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
		var s models.Work
		var imagesJSON []byte
		var videosJSON []byte
		var languagesJSON []byte
		var likedStr string
		var price, priceTo sql.NullFloat64
		var negotiable bool
		var hidePhone bool
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &negotiable, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &likedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &hidePhone, &s.CreatedAt,

			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan error: %w", err)
		}

		s.Price = floatFromNull(price)
		s.PriceTo = floatFromNull(priceTo)
		s.Negotiable = negotiable
		s.HidePhone = hidePhone

		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
		}

		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode videos error: %w", err)
			}
		}

		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode languages error: %w", err)
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

	liftListingsTopOnly(works, func(w models.Work) string { return w.Top })

	// Get min/max prices
	var minPrice, maxPrice sql.NullFloat64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM work`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return works, 0, 0, nil // fallback
	}

	return works, minPrice.Float64, maxPrice.Float64, nil
}

func (r *WorkRepository) GetWorksByUserID(ctx context.Context, userID int) ([]models.Work, error) {
	query := `
             SELECT s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.hide_phone, s.created_at, s.updated_at
FROM work s
JOIN users u ON s.user_id = u.id
LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
WHERE s.user_id = ?
`

	rows, err := r.DB.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var s models.Work
		var imagesJSON []byte
		var videosJSON []byte
		var languagesJSON []byte
		var price, priceTo sql.NullFloat64
		var likedStr string
		var negotiable bool
		var hidePhone bool
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &negotiable, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &likedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &hidePhone, &s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		s.Price = floatFromNull(price)
		s.PriceTo = floatFromNull(priceTo)
		s.Negotiable = negotiable
		s.HidePhone = hidePhone

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

		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("json decode languages error: %w", err)
			}
		}

		s.Liked = likedStr == "1"
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

s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.description, s.work_experience, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.hide_phone, s.created_at
FROM work s
JOIN users u ON s.user_id = u.id
WHERE 1=1 AND s.status != 'archive'
`
	args := []interface{}{}

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

	if req.NegotiableOnly {
		query += " AND s.negotiable = 1"
	}

	if len(req.WorkExperiences) > 0 {
		placeholders := strings.Repeat("?,", len(req.WorkExperiences))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.work_experience IN (%s)", placeholders)
		for _, item := range req.WorkExperiences {
			args = append(args, item)
		}
	}

	if len(req.Schedules) > 0 {
		placeholders := strings.Repeat("?,", len(req.Schedules))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.schedule IN (%s)", placeholders)
		for _, item := range req.Schedules {
			args = append(args, item)
		}
	}

	if len(req.PaymentPeriods) > 0 {
		placeholders := strings.Repeat("?,", len(req.PaymentPeriods))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.payment_period IN (%s)", placeholders)
		for _, item := range req.PaymentPeriods {
			args = append(args, item)
		}
	}

	if len(req.RemoteOptions) > 0 {
		placeholders := strings.Repeat("?,", len(req.RemoteOptions))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.distance_work IN (%s)", placeholders)
		for _, item := range req.RemoteOptions {
			args = append(args, item)
		}
	}

	if len(req.Educations) > 0 {
		placeholders := strings.Repeat("?,", len(req.Educations))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.education IN (%s)", placeholders)
		for _, item := range req.Educations {
			args = append(args, item)
		}
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

	radiusKm := req.RadiusKm
	if req.Nearby && radiusKm == nil && req.Latitude != nil && req.Longitude != nil {
		defaultRadius := 5.0
		radiusKm = &defaultRadius
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
		var languagesJSON []byte
		var price, priceTo sql.NullFloat64
		var negotiable bool
		var hidePhone bool
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,
			&s.WorkID, &s.WorkName, &s.WorkAddress, &price, &priceTo, &negotiable, &s.WorkDescription, &s.WorkExperience, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &hidePhone, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.WorkPrice = floatFromNull(price)
		s.WorkPriceTo = floatFromNull(priceTo)
		s.Negotiable = negotiable
		s.HidePhone = hidePhone
		var latPtr, lonPtr *string
		if lat.Valid {
			s.WorkLatitude = lat.String
			latPtr = &s.WorkLatitude
		}
		if lon.Valid {
			s.WorkLongitude = lon.String
			lonPtr = &s.WorkLongitude
		}
		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, latPtr, lonPtr)
		if radiusKm != nil && (s.Distance == nil || *s.Distance > *radiusKm) {
			continue
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}
		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("failed to decode languages json: %w", err)
			}
		}
		if len(req.Languages) > 0 && !matchesLanguageFilter(req.Languages, s.Languages) {
			continue
		}
		if req.TwentyFourSeven && !isRoundTheClock(s.WorkTimeFrom, s.WorkTimeTo) {
			continue
		}
		if req.OpenNow && !isCurrentlyOpen(s.WorkTimeFrom, s.WorkTimeTo, time.Now()) {
			continue
		}
		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.UserReviewsCount = count
		}
		works = append(works, s)
	}

	liftListingsTopOnly(works, func(w models.FilteredWork) string { return w.Top })
	return works, nil
}

func (r *WorkRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Work, error) {
	query := `
SELECT
s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.user_id,
u.id, u.name, u.surname, u.review_rating, u.avatar_path,
s.images, s.videos, s.category_id, s.subcategory_id, s.description,
s.avg_rating, s.top, s.liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.hide_phone,
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
		var languagesJSON []byte
		var price, priceTo sql.NullFloat64
		var negotiable bool
		var hidePhone bool
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &negotiable, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &hidePhone,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		s.Price = floatFromNull(price)
		s.PriceTo = floatFromNull(priceTo)
		s.Negotiable = negotiable
		s.HidePhone = hidePhone
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("json decode error: %w", err)
		}
		if len(videosJSON) > 0 {
			if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
				return nil, fmt.Errorf("json decode videos error: %w", err)
			}
		}
		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("json decode languages error: %w", err)
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

           s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.description, s.work_experience, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
       COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
       s.top, s.hide_phone, s.created_at,
       CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
FROM work s
JOIN users u ON s.user_id = u.id
LEFT JOIN work_favorites sf ON sf.work_id = s.id AND sf.user_id = ?
LEFT JOIN work_responses sr ON sr.work_id = s.id AND sr.user_id = ?
WHERE 1=1 AND s.status != 'archive'
`

	args := []interface{}{userID, userID}

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

	if req.NegotiableOnly {
		query += " AND s.negotiable = 1"
	}

	if len(req.WorkExperiences) > 0 {
		placeholders := strings.Repeat("?,", len(req.WorkExperiences))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.work_experience IN (%s)", placeholders)
		for _, item := range req.WorkExperiences {
			args = append(args, item)
		}
	}

	if len(req.Schedules) > 0 {
		placeholders := strings.Repeat("?,", len(req.Schedules))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.schedule IN (%s)", placeholders)
		for _, item := range req.Schedules {
			args = append(args, item)
		}
	}

	if len(req.PaymentPeriods) > 0 {
		placeholders := strings.Repeat("?,", len(req.PaymentPeriods))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.payment_period IN (%s)", placeholders)
		for _, item := range req.PaymentPeriods {
			args = append(args, item)
		}
	}

	if len(req.RemoteOptions) > 0 {
		placeholders := strings.Repeat("?,", len(req.RemoteOptions))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.distance_work IN (%s)", placeholders)
		for _, item := range req.RemoteOptions {
			args = append(args, item)
		}
	}

	if len(req.Educations) > 0 {
		placeholders := strings.Repeat("?,", len(req.Educations))
		placeholders = placeholders[:len(placeholders)-1]
		query += fmt.Sprintf(" AND s.education IN (%s)", placeholders)
		for _, item := range req.Educations {
			args = append(args, item)
		}
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

	radiusKm := req.RadiusKm
	if req.Nearby && radiusKm == nil && req.Latitude != nil && req.Longitude != nil {
		defaultRadius := 5.0
		radiusKm = &defaultRadius
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
		var languagesJSON []byte
		var likedStr, respondedStr string
		var price, priceTo sql.NullFloat64
		var negotiable bool
		var hidePhone bool
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.WorkID, &s.WorkName, &s.WorkAddress, &price, &priceTo, &negotiable, &s.WorkDescription, &s.WorkExperience, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &lat, &lon, &imagesJSON, &videosJSON, &s.Top, &hidePhone, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		s.WorkPrice = floatFromNull(price)
		s.WorkPriceTo = floatFromNull(priceTo)
		s.Negotiable = negotiable
		s.HidePhone = hidePhone
		var latPtr, lonPtr *string
		if lat.Valid {
			s.WorkLatitude = lat.String
			latPtr = &s.WorkLatitude
		}
		if lon.Valid {
			s.WorkLongitude = lon.String
			lonPtr = &s.WorkLongitude
		}
		s.Distance = calculateDistanceKm(req.Latitude, req.Longitude, latPtr, lonPtr)
		if radiusKm != nil && (s.Distance == nil || *s.Distance > *radiusKm) {
			continue
		}
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return nil, fmt.Errorf("failed to decode images json: %w", err)
		}
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return nil, fmt.Errorf("failed to decode videos json: %w", err)
		}
		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("failed to decode languages json: %w", err)
			}
		}
		s.Liked = likedStr == "1"
		s.Responded = respondedStr == "1"
		if len(req.Languages) > 0 && !matchesLanguageFilter(req.Languages, s.Languages) {
			continue
		}
		if req.TwentyFourSeven && !isRoundTheClock(s.WorkTimeFrom, s.WorkTimeTo) {
			continue
		}
		if req.OpenNow && !isCurrentlyOpen(s.WorkTimeFrom, s.WorkTimeTo, time.Now()) {
			continue
		}
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

	liftListingsTopOnly(works, func(w models.FilteredWork) string { return w.Top })
	log.Printf("[INFO] Successfully fetched %d services", len(works))
	return works, nil
}

func (r *WorkRepository) GetWorkByWorkIDAndUserID(ctx context.Context, workID int, userID int) (models.Work, error) {
	query := `
    SELECT
s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.user_id,
u.id, u.name, u.surname, u.review_rating, u.avatar_path,
  CASE WHEN sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
   s.images, s.videos, s.category_id, c.name,
   s.subcategory_id, sub.name, sub.name_kz,
   s.description, s.avg_rating, s.top,
      CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
      CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
      s.status, s.work_experience, u.city_id, city.name, city.type, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.hide_phone, s.created_at, s.updated_at, s.work_time_from, s.work_time_to
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
	var languagesJSON []byte

	var likedStr, respondedStr string
	var price, priceTo sql.NullFloat64
	var negotiable bool
	var hidePhone bool

	err := r.DB.QueryRowContext(ctx, query, userID, userID, workID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &negotiable, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName, &s.SubcategoryNameKz,
		&s.Description, &s.AvgRating, &s.Top,
		&likedStr, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &hidePhone, &s.CreatedAt, &s.UpdatedAt, &s.WorkTimeFrom, &s.WorkTimeTo,
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
	if len(languagesJSON) > 0 {
		if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
			return models.Work{}, fmt.Errorf("failed to decode languages json: %w", err)
		}
	}

	s.Price = floatFromNull(price)
	s.PriceTo = floatFromNull(priceTo)
	s.Negotiable = negotiable
	s.HidePhone = hidePhone
	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"
	if s.HidePhone {
		s.User.Phone = ""
	}

	s.AvgRating = getAverageRating(ctx, r.DB, "work_reviews", "work_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

func nullableFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func floatFromNull(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	return &n.Float64
}

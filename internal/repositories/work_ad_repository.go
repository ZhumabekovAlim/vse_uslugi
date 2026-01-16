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
	ErrWorkAdNotFound = errors.New("service not found")
)

type WorkAdRepository struct {
	DB *sql.DB
}

func (r *WorkAdRepository) CreateWorkAd(ctx context.Context, work models.WorkAd) (models.WorkAd, error) {
	query := `
        INSERT INTO work_ad (name, address, price, price_to, negotiable, hide_phone, user_id, images, videos, category_id, subcategory_id, description, avg_rating, top, liked, status, work_experience, city_id, schedule, distance_work, payment_period, languages, education, first_name, last_name, birth_date, contact_number, work_time_from, work_time_to, latitude, longitude, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.WorkAd{}, err
	}

	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.WorkAd{}, err
	}

	languagesJSON, err := json.Marshal(work.Languages)
	if err != nil {
		return models.WorkAd{}, err
	}

	var subcategory interface{}
	if work.SubcategoryID != 0 {
		subcategory = work.SubcategoryID
	}

	result, err := r.DB.ExecContext(ctx, query,
		work.Name,
		work.Address,
		work.Price,
		work.PriceTo,
		work.Negotiable,
		work.HidePhone,
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
		work.FirstName,
		work.LastName,
		work.BirthDate,
		work.ContactNumber,
		work.WorkTimeFrom,
		work.WorkTimeTo,
		work.Latitude,
		work.Longitude,
		work.CreatedAt,
	)
	if err != nil {
		return models.WorkAd{}, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return models.WorkAd{}, err
	}
	work.ID = int(lastID)
	return work, nil
}

func (r *WorkAdRepository) GetWorkAdByID(ctx context.Context, id int, userID int) (models.WorkAd, error) {
	query := `
     SELECT w.id, w.name, w.address, w.price, w.price_to, w.negotiable, w.hide_phone, w.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, w.description, w.avg_rating, w.top,
            CASE WHEN sf.work_ad_id IS NOT NULL THEN '1' ELSE '0' END AS liked,

             CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,

             w.status, w.work_experience, u.city_id, city.name, city.type, w.schedule, w.distance_work, w.payment_period, w.languages, w.education, w.first_name, w.last_name, w.birth_date, w.contact_number, w.work_time_from, w.work_time_to, w.latitude, w.longitude, w.created_at, w.updated_at
       FROM work_ad w
       JOIN users u ON w.user_id = u.id
       JOIN work_categories c ON w.category_id = c.id
       JOIN work_subcategories sub ON w.subcategory_id = sub.id
       JOIN cities city ON u.city_id = city.id
       LEFT JOIN work_ad_responses sr ON sr.work_ad_id = w.id AND sr.user_id = ?
       LEFT JOIN work_ad_favorites sf ON sf.work_ad_id = w.id AND sf.user_id = ?
       WHERE w.id = ? AND w.status <> 'archive'
`

	var s models.WorkAd
	var imagesJSON []byte
	var videosJSON []byte
	var languagesJSON []byte
	var respondedStr string

	var price, priceTo sql.NullFloat64

	err := r.DB.QueryRowContext(ctx, query, userID, userID, id).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &s.CreatedAt,

		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.WorkAd{}, errors.New("not found")
	}
	if err != nil {
		return models.WorkAd{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	if len(languagesJSON) > 0 {
		if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode languages json: %w", err)
		}
	}

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}

	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *WorkAdRepository) GetWorkAdByIDWithCity(ctx context.Context, id int, userID int, cityID int) (models.WorkAd, error) {
	query := `
     SELECT w.id, w.name, w.address, w.price, w.price_to, w.negotiable, w.hide_phone, w.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, w.images, w.videos, w.category_id, c.name, w.subcategory_id, sub.name, w.description, w.avg_rating, w.top, w.liked,

             CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,

             w.status, w.work_experience, u.city_id, city.name, city.type, w.schedule, w.distance_work, w.payment_period, w.languages, w.education, w.first_name, w.last_name, w.birth_date, w.contact_number, w.work_time_from, w.work_time_to, w.latitude, w.longitude, w.created_at, w.updated_at
       FROM work_ad w
       JOIN users u ON w.user_id = u.id
       JOIN work_categories c ON w.category_id = c.id
       JOIN work_subcategories sub ON w.subcategory_id = sub.id
       JOIN cities city ON u.city_id = city.id
       LEFT JOIN work_ad_responses sr ON sr.work_ad_id = w.id AND sr.user_id = ?
       WHERE w.id = ? AND w.status <> 'archive' AND w.city_id = ?
`

	var s models.WorkAd
	var imagesJSON []byte
	var videosJSON []byte
	var languagesJSON []byte
	var respondedStr string

	var price, priceTo sql.NullFloat64

	err := r.DB.QueryRowContext(ctx, query, userID, id, cityID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName, &s.SubcategoryID, &s.SubcategoryName, &s.Description, &s.AvgRating, &s.Top, &s.Liked, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &s.CreatedAt,

		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.WorkAd{}, errors.New("not found")
	}
	if err != nil {
		return models.WorkAd{}, err
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	if len(languagesJSON) > 0 {
		if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode languages json: %w", err)
		}
	}

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}

	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}
	return s, nil
}

func (r *WorkAdRepository) UpdateWorkAd(ctx context.Context, work models.WorkAd) (models.WorkAd, error) {
	query := `
        UPDATE work_ad
        SET name = ?, address = ?, price = ?, price_to = ?, negotiable = ?, hide_phone = ?, user_id = ?, images = ?, videos = ?, category_id = ?, subcategory_id = ?,
            description = ?, avg_rating = ?, top = ?, liked = ?, status = ?, work_experience = ?, city_id = ?, schedule = ?, distance_work = ?, payment_period = ?, languages = ?, education = ?, first_name = ?, last_name = ?, birth_date = ?, contact_number = ?, work_time_from = ?, work_time_to = ?, latitude = ?, longitude = ?, updated_at = ?
        WHERE id = ?
    `
	imagesJSON, err := json.Marshal(work.Images)
	if err != nil {
		return models.WorkAd{}, fmt.Errorf("failed to marshal images: %w", err)
	}

	videosJSON, err := json.Marshal(work.Videos)
	if err != nil {
		return models.WorkAd{}, fmt.Errorf("failed to marshal videos: %w", err)
	}
	languagesJSON, err := json.Marshal(work.Languages)
	if err != nil {
		return models.WorkAd{}, fmt.Errorf("failed to marshal languages: %w", err)
	}
	updatedAt := time.Now()
	work.UpdatedAt = &updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		work.Name, work.Address, work.Price, work.PriceTo, work.Negotiable, work.HidePhone, work.UserID, imagesJSON, videosJSON,
		work.CategoryID, work.SubcategoryID, work.Description, work.AvgRating, work.Top, work.Liked, work.Status, work.WorkExperience, work.CityID, work.Schedule, work.DistanceWork, work.PaymentPeriod, languagesJSON, work.Education, work.FirstName, work.LastName, work.BirthDate, work.ContactNumber, work.WorkTimeFrom, work.WorkTimeTo, work.Latitude, work.Longitude, work.UpdatedAt, work.ID,
	)
	if err != nil {
		return models.WorkAd{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.WorkAd{}, err
	}
	if rowsAffected == 0 {
		return models.WorkAd{}, ErrWorkAdNotFound
	}
	return r.GetWorkAdByID(ctx, work.ID, 0)
}

func (r *WorkAdRepository) DeleteWorkAd(ctx context.Context, id int) error {
	query := `DELETE FROM work_ad WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWorkAdNotFound
	}
	return nil
}

func (r *WorkAdRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE work_ad SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.DB.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkAdNotFound
	}
	return nil
}

func (r *WorkAdRepository) GetStatus(ctx context.Context, id int) (string, error) {
	var status string
	if err := r.DB.QueryRowContext(ctx, `SELECT status FROM work_ad WHERE id = ?`, id).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return "", ErrWorkAdNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *WorkAdRepository) ArchiveByUserID(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE work_ad SET status = 'archive', updated_at = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}
func (r *WorkAdRepository) GetWorksAdWithFilters(ctx context.Context, userID int, cityID int, categories []int, subcategories []string, priceFrom, priceTo float64, ratings []float64, sortOption, limit, offset int, negotiable *bool, experience []string, schedules []string, paymentPeriods []string, remoteWork *bool, languages []string, educations []string) ([]models.WorkAd, float64, float64, error) {
	var (
		works_ad   []models.WorkAd
		params     []interface{}
		conditions []string
	)

	baseQuery := `

       SELECT s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id, u.id, u.name, u.surname, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, CASE WHEN sf.work_ad_id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.status, s.work_experience, s.city_id, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.first_name, s.last_name, s.birth_date, s.contact_number, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.created_at, s.updated_at

               FROM work_ad s
               LEFT JOIN work_ad_favorites sf ON sf.work_ad_id = s.id AND sf.user_id = ?
               JOIN users u ON s.user_id = u.id
               INNER JOIN work_categories c ON s.category_id = c.id

       `
	params = append(params, userID)

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
		var s models.WorkAd
		var imagesJSON []byte
		var videosJSON []byte
		var languagesJSON []byte
		var likedStr string
		var price, priceTo sql.NullFloat64
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID, &s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,

			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &likedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &s.CreatedAt,

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
				return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
			}
		}

		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, 0, 0, fmt.Errorf("json decode error: %w", err)
			}
		}

		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}

		s.Liked = likedStr == "1"

		s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)

		count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
		if err == nil {
			s.User.ReviewsCount = count
		}
		works_ad = append(works_ad, s)
	}

	liftListingsTopOnly(works_ad, func(w models.WorkAd) string { return w.Top })

	// Get min/max prices
	var minPrice, maxPrice float64
	err = r.DB.QueryRowContext(ctx, `SELECT MIN(price), MAX(price) FROM work_ad`).Scan(&minPrice, &maxPrice)
	if err != nil {
		return works_ad, 0, 0, nil // fallback
	}

	return works_ad, minPrice, maxPrice, nil
}

func (r *WorkAdRepository) GetWorksAdByUserID(ctx context.Context, userID int) ([]models.WorkAd, error) {
	query := `
                SELECT s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id, u.id, u.name, u.review_rating, u.avatar_path, s.images, s.videos, s.category_id, s.subcategory_id, s.description, s.avg_rating, s.top, CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.first_name, s.last_name, s.birth_date, s.contact_number, s.work_time_from, s.work_time_to, s.latitude, s.longitude, s.created_at, s.updated_at
                FROM work_ad s
                JOIN users u ON s.user_id = u.id
                LEFT JOIN work_ad_favorites sf ON sf.work_ad_id = s.id AND sf.user_id = ?
                WHERE s.user_id = ?
       `

	rows, err := r.DB.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works_ad []models.WorkAd
	for rows.Next() {
		var s models.WorkAd
		var imagesJSON []byte
		var videosJSON []byte
		var languagesJSON []byte
		var price, priceTo sql.NullFloat64
		var likedStr string
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID, &s.User.ID, &s.User.Name, &s.User.ReviewRating, &s.User.AvatarPath, &imagesJSON, &videosJSON,
			&s.CategoryID, &s.SubcategoryID, &s.Description, &s.AvgRating, &s.Top, &likedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude, &s.CreatedAt,
			&s.UpdatedAt,
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

		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}

		s.Liked = likedStr == "1"
		s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)

		works_ad = append(works_ad, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortWorkAdsByTop(works_ad)

	return works_ad, nil
}

func (r *WorkAdRepository) GetFilteredWorksAdPost(ctx context.Context, req models.FilterWorkAdRequest) ([]models.FilteredWorkAd, error) {
	query := `
SELECT

u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.description, s.work_experience, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.first_name, s.last_name, s.birth_date, s.contact_number, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
s.top, s.created_at
FROM work_ad s
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

	var works []models.FilteredWorkAd
	for rows.Next() {
		var s models.FilteredWorkAd
		var imagesJSON, videosJSON, languagesJSON []byte
		var price, priceTo sql.NullFloat64
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,
			&s.WorkAdID, &s.WorkAdName, &s.WorkAdAddress, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.WorkAdDescription, &s.WorkExperience, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.WorkAdLatitude, &s.WorkAdLongitude,
			&imagesJSON, &videosJSON, &s.Top, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		if price.Valid {
			s.WorkAdPrice = &price.Float64
		}
		if priceTo.Valid {
			s.WorkAdPriceTo = &priceTo.Float64
		}
		var latPtr, lonPtr *string
		if s.WorkAdLatitude != "" {
			latPtr = &s.WorkAdLatitude
		}
		if s.WorkAdLongitude != "" {
			lonPtr = &s.WorkAdLongitude
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

	liftListingsTopOnly(works, func(w models.FilteredWorkAd) string { return w.Top })
	return works, nil
}

func (r *WorkAdRepository) FetchByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.WorkAd, error) {
	query := `
        SELECT
                s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id,
                u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                s.images, s.videos, s.category_id, s.subcategory_id, s.description,
                s.avg_rating, s.top, s.liked, s.status, s.work_experience, u.city_id, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.first_name, s.last_name, s.birth_date, s.contact_number, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
                s.created_at, s.updated_at
        FROM work_ad s
        JOIN users u ON s.user_id = u.id
        WHERE s.status = ? AND s.user_id = ?`

	rows, err := r.DB.QueryContext(ctx, query, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.WorkAd
	for rows.Next() {
		var s models.WorkAd
		var imagesJSON []byte
		var videosJSON []byte
		var languagesJSON []byte
		var price, priceTo sql.NullFloat64
		err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID,
			&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath,
			&imagesJSON, &videosJSON, &s.CategoryID, &s.SubcategoryID,
			&s.Description, &s.AvgRating, &s.Top, &s.Liked, &s.Status, &s.WorkExperience, &s.CityID, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Latitude, &s.Longitude,
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
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}
		if len(languagesJSON) > 0 {
			if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
		}
		if price.Valid {
			s.Price = &price.Float64
		}
		if priceTo.Valid {
			s.PriceTo = &priceTo.Float64
		}
		s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)
		works = append(works, s)
	}
	sortWorkAdsByTop(works)
	return works, nil
}

func (r *WorkAdRepository) GetFilteredWorksAdWithLikes(ctx context.Context, req models.FilterWorkAdRequest, userID int) ([]models.FilteredWorkAd, error) {
	log.Printf("[INFO] Start GetFilteredServicesWithLikes for user_id=%d", userID)

	query := `
    SELECT DISTINCT

           u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0),

           s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.description, s.work_experience, s.schedule, s.distance_work, s.payment_period, s.languages, s.education, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
       COALESCE(s.images, '[]') AS images, COALESCE(s.videos, '[]') AS videos,
       s.top, s.created_at,
           CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
           CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded
   FROM work_ad s
   JOIN users u ON s.user_id = u.id
   LEFT JOIN work_ad_favorites sf ON sf.work_ad_id = s.id AND sf.user_id = ?
  LEFT JOIN work_ad_responses sr ON sr.work_ad_id = s.id AND sr.user_id = ?
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

	var works []models.FilteredWorkAd
	for rows.Next() {
		var s models.FilteredWorkAd
		var imagesJSON, videosJSON []byte
		var likedStr, respondedStr string
		var price, priceTo sql.NullFloat64
		var languagesJSON []byte
		if err := rows.Scan(
			&s.UserID, &s.UserName, &s.UserSurname, &s.UserAvatarPath, &s.UserRating,

			&s.WorkAdID, &s.WorkAdName, &s.WorkAdAddress, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.WorkAdDescription, &s.WorkExperience, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &languagesJSON, &s.Education, &s.WorkTimeFrom, &s.WorkTimeTo, &s.WorkAdLatitude, &s.WorkAdLongitude, &imagesJSON, &videosJSON, &s.Top, &s.CreatedAt, &likedStr, &respondedStr,
		); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if price.Valid {
			s.WorkAdPrice = &price.Float64
		}
		if priceTo.Valid {
			s.WorkAdPriceTo = &priceTo.Float64
		}
		var latPtr, lonPtr *string
		if s.WorkAdLatitude != "" {
			latPtr = &s.WorkAdLatitude
		}
		if s.WorkAdLongitude != "" {
			lonPtr = &s.WorkAdLongitude
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

	liftListingsTopOnly(works, func(w models.FilteredWorkAd) string { return w.Top })
	log.Printf("[INFO] Successfully fetched %d services", len(works))
	return works, nil
}

func (r *WorkAdRepository) GetWorkAdByWorkIDAndUserID(ctx context.Context, workadID int, userID int) (models.WorkAd, error) {
	query := `
            SELECT
                    s.id, s.name, s.address, s.price, s.price_to, s.negotiable, s.hide_phone, s.user_id,
                    u.id, u.name, u.surname, u.review_rating, u.avatar_path,
                      CASE WHEN sr.id IS NOT NULL THEN u.phone ELSE '' END AS phone,
                       s.images, s.videos, s.category_id, c.name,
                       s.subcategory_id, sub.name,
                       s.description, s.avg_rating, s.top,
              CASE WHEN sf.id IS NOT NULL THEN '1' ELSE '0' END AS liked,
              CASE WHEN sr.id IS NOT NULL THEN '1' ELSE '0' END AS responded,
              s.status, s.work_experience, u.city_id, city.name, city.type, s.schedule, s.distance_work, s.payment_period, s.latitude, s.longitude, s.created_at, s.updated_at, s.work_time_from, s.work_time_to, s.education, s.first_name, s.last_name, s.birth_date, s.contact_number, s.languages
                FROM work_ad s
                JOIN users u ON s.user_id = u.id
                JOIN work_categories c ON s.category_id = c.id
                JOIN work_subcategories sub ON s.subcategory_id = sub.id
                JOIN cities city ON u.city_id = city.id
                LEFT JOIN work_ad_favorites sf ON sf.work_ad_id = s.id AND sf.user_id = ?
                LEFT JOIN work_ad_responses sr ON sr.work_ad_id = s.id AND sr.user_id = ?
                WHERE s.id = ?
       `

	var s models.WorkAd
	var imagesJSON []byte
	var videosJSON []byte
	var languagesJSON []byte

	var likedStr, respondedStr string

	var price, priceTo sql.NullFloat64

	err := r.DB.QueryRowContext(ctx, query, userID, userID, workadID).Scan(
		&s.ID, &s.Name, &s.Address, &price, &priceTo, &s.Negotiable, &s.HidePhone, &s.UserID,
		&s.User.ID, &s.User.Name, &s.User.Surname, &s.User.ReviewRating, &s.User.AvatarPath, &s.User.Phone,
		&imagesJSON, &videosJSON, &s.CategoryID, &s.CategoryName,
		&s.SubcategoryID, &s.SubcategoryName,
		&s.Description, &s.AvgRating, &s.Top,
		&likedStr, &respondedStr, &s.Status, &s.WorkExperience, &s.CityID, &s.CityName, &s.CityType, &s.Schedule, &s.DistanceWork, &s.PaymentPeriod, &s.Latitude, &s.Longitude, &s.CreatedAt, &s.UpdatedAt, &s.WorkTimeFrom, &s.WorkTimeTo, &s.Education, &s.FirstName, &s.LastName, &s.BirthDate, &s.ContactNumber, &languagesJSON,
	)

	if err == sql.ErrNoRows {
		return models.WorkAd{}, errors.New("service not found")
	}
	if err != nil {
		return models.WorkAd{}, fmt.Errorf("failed to get service: %w", err)
	}

	if len(imagesJSON) > 0 {
		if err := json.Unmarshal(imagesJSON, &s.Images); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode images json: %w", err)
		}
	}

	if len(videosJSON) > 0 {
		if err := json.Unmarshal(videosJSON, &s.Videos); err != nil {
			return models.WorkAd{}, fmt.Errorf("failed to decode videos json: %w", err)
		}
	}

	if len(languagesJSON) > 0 {
		if err := json.Unmarshal(languagesJSON, &s.Languages); err != nil {
			return models.WorkAd{}, fmt.Errorf("json decode error: %w", err)
		}
	}

	if price.Valid {
		s.Price = &price.Float64
	}
	if priceTo.Valid {
		s.PriceTo = &priceTo.Float64
	}

	s.Liked = likedStr == "1"
	s.Responded = respondedStr == "1"

	s.AvgRating = getAverageRating(ctx, r.DB, "work_ad_reviews", "work_ad_id", s.ID)

	count, err := getUserTotalReviews(ctx, r.DB, s.UserID)
	if err == nil {
		s.User.ReviewsCount = count
	}

	return s, nil
}

package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

// UserItemsRepository retrieves user items across multiple entities.
type UserItemsRepository struct {
	DB *sql.DB
}

// GetServiceWorkRentByUserID returns service, work and rent items owned by the user ordered by creation time.
func (r *UserItemsRepository) GetServiceWorkRentByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT id, name, price, description, created_at, status, 'service' AS type FROM service WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, status, 'work' AS type FROM work WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, status, 'rent' AS type FROM rent WHERE user_id = ?
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		var price sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Name, &price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		if price.Valid {
			item.Price = &price.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetAdWorkAdRentAdByUserID returns ad, work_ad and rent_ad items owned by the user ordered by creation time.
func (r *UserItemsRepository) GetAdWorkAdRentAdByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT id, name, price, description, created_at, status, 'ad' AS type FROM ad WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, status, 'work_ad' AS type FROM work_ad WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, status, 'rent_ad' AS type FROM rent_ad WHERE user_id = ?
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		var price sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Name, &price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		if price.Valid {
			item.Price = &price.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetOrderHistoryByUserID returns completed service, work, rent, ad, work_ad and rent_ad items for the user ordered by creation time.
func (r *UserItemsRepository) GetOrderHistoryByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
SELECT id, name, address, city_name, price, description, created_at, status, type FROM (
SELECT s.id, s.name, s.address, city.name AS city_name, s.price, s.description, sc.created_at, sc.status, 'service' AS type
FROM service_confirmations sc
JOIN service s ON s.id = sc.service_id
LEFT JOIN cities city ON s.city_id = city.id
WHERE (sc.performer_id = ? OR sc.client_id = ?) AND sc.status IN ('in_progress', 'archived', 'done')
        UNION ALL
SELECT w.id, w.name, w.address, city.name AS city_name, w.price, w.description, wc.created_at, wc.status, 'work' AS type
FROM work_confirmations wc
JOIN work w ON w.id = wc.work_id
LEFT JOIN cities city ON w.city_id = city.id
WHERE (wc.performer_id = ? OR wc.client_id = ?) AND wc.status IN ('in_progress', 'archived', 'done')
        UNION ALL
SELECT r.id, r.name, r.address, city.name AS city_name, r.price, r.description, rc.created_at, rc.status, 'rent' AS type
FROM rent_confirmations rc
JOIN rent r ON r.id = rc.rent_id
LEFT JOIN cities city ON r.city_id = city.id
WHERE (rc.performer_id = ? OR rc.client_id = ?) AND rc.status IN ('in_progress', 'archived', 'done')
        UNION ALL
SELECT a.id, a.name, a.address, city.name AS city_name, a.price, a.description, ac.created_at, ac.status, 'ad' AS type
FROM ad_confirmations ac
JOIN ad a ON a.id = ac.ad_id
LEFT JOIN cities city ON a.city_id = city.id
WHERE (ac.performer_id = ? OR ac.client_id = ?) AND ac.status IN ('in_progress', 'archived', 'done')
        UNION ALL
SELECT wa.id, wa.name, wa.address, city.name AS city_name, wa.price, wa.description, wac.created_at, wac.status, 'work_ad' AS type
FROM work_ad_confirmations wac
JOIN work_ad wa ON wa.id = wac.work_ad_id
LEFT JOIN cities city ON wa.city_id = city.id
WHERE (wac.performer_id = ? OR wac.client_id = ?) AND wac.status IN ('in_progress', 'archived', 'done')
        UNION ALL
SELECT ra.id, ra.name, ra.address, city.name AS city_name, ra.price, ra.description, rac.created_at, rac.status, 'rent_ad' AS type
FROM rent_ad_confirmations rac
JOIN rent_ad ra ON ra.id = rac.rent_ad_id
LEFT JOIN cities city ON ra.city_id = city.id
WHERE (rac.performer_id = ? OR rac.client_id = ?) AND rac.status IN ('in_progress', 'archived', 'done')
    ) AS combined
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query,
		userID, userID,
		userID, userID,
		userID, userID,
		userID, userID,
		userID, userID,
		userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		var price sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Name, &item.Address, &item.CityName, &price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		if price.Valid {
			item.Price = &price.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetActiveOrdersByUserID returns all orders with status active where the user is the performer.
func (r *UserItemsRepository) GetActiveOrdersByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT s.id, s.name, s.price, s.description, sc.created_at, sc.status, 'service' AS type
    FROM service_confirmations sc
    JOIN service s ON s.id = sc.service_id
    WHERE sc.performer_id = ? AND sc.confirmed = true AND sc.status = 'active'
    UNION ALL
    SELECT w.id, w.name, w.price, w.description, wc.created_at, wc.status, 'work' AS type
    FROM work_confirmations wc
    JOIN work w ON w.id = wc.work_id
    WHERE wc.performer_id = ? AND wc.confirmed = true AND wc.status = 'active'
    UNION ALL
    SELECT r.id, r.name, r.price, r.description, rc.created_at, rc.status, 'rent' AS type
    FROM rent_confirmations rc
    JOIN rent r ON r.id = rc.rent_id
    WHERE rc.performer_id = ? AND rc.confirmed = true AND rc.status = 'active'
    UNION ALL
    SELECT a.id, a.name, a.price, a.description, ac.created_at, ac.status, 'ad' AS type
    FROM ad_confirmations ac
    JOIN ad a ON a.id = ac.ad_id
    WHERE ac.performer_id = ? AND ac.confirmed = true AND ac.status = 'active'
    UNION ALL
    SELECT wa.id, wa.name, wa.price, wa.description, wac.created_at, wac.status, 'work_ad' AS type
    FROM work_ad_confirmations wac
    JOIN work_ad wa ON wa.id = wac.work_ad_id
    WHERE wac.performer_id = ? AND wac.confirmed = true AND wac.status = 'active'
    UNION ALL
    SELECT ra.id, ra.name, ra.price, ra.description, rac.created_at, rac.status, 'rent_ad' AS type
    FROM rent_ad_confirmations rac
    JOIN rent_ad ra ON ra.id = rac.rent_ad_id
    WHERE rac.performer_id = ? AND rac.confirmed = true AND rac.status = 'active'
    ORDER BY created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		var price sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Name, &price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		if price.Valid {
			item.Price = &price.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

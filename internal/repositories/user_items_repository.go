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
    SELECT id, name, price, description, created_at, 'service' AS type FROM service WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, 'work' AS type FROM work WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, 'rent' AS type FROM rent WHERE user_id = ?
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetAdWorkAdRentAdByUserID returns ad, work_ad and rent_ad items owned by the user ordered by creation time.
func (r *UserItemsRepository) GetAdWorkAdRentAdByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT id, name, price, description, created_at, 'ad' AS type FROM ad WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, 'work_ad' AS type FROM work_ad WHERE user_id = ?
    UNION ALL
    SELECT id, name, price, description, created_at, 'rent_ad' AS type FROM rent_ad WHERE user_id = ?
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetOrderHistoryByUserID returns completed service, work, rent, ad, work_ad and rent_ad items for the user ordered by creation time.
func (r *UserItemsRepository) GetOrderHistoryByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT id, name, price, description, created_at, 'service' AS type FROM service WHERE user_id = ? AND status = 'done'
    UNION ALL
    SELECT id, name, price, description, created_at, 'work' AS type FROM work WHERE user_id = ? AND status = 'done'
    UNION ALL
    SELECT id, name, price, description, created_at, 'rent' AS type FROM rent WHERE user_id = ? AND status = 'done'
    UNION ALL
    SELECT id, name, price, description, created_at, 'ad' AS type FROM ad WHERE user_id = ? AND status = 'done'
    UNION ALL
    SELECT id, name, price, description, created_at, 'work_ad' AS type FROM work_ad WHERE user_id = ? AND status = 'done'
    UNION ALL
    SELECT id, name, price, description, created_at, 'rent_ad' AS type FROM rent_ad WHERE user_id = ? AND status = 'done'
    ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetActiveOrdersByUserID returns all active orders where the user is the performer.
func (r *UserItemsRepository) GetActiveOrdersByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT s.id, s.name, s.price, s.description, s.created_at, 'service' AS type
    FROM service_confirmations sc
    JOIN service s ON s.id = sc.service_id
    WHERE sc.performer_id = ? AND sc.confirmed = true AND s.status = 'active'
    UNION ALL
    SELECT w.id, w.name, w.price, w.description, w.created_at, 'work' AS type
    FROM work_confirmations wc
    JOIN work w ON w.id = wc.work_id
    WHERE wc.performer_id = ? AND wc.confirmed = true AND w.status = 'active'
    UNION ALL
    SELECT r.id, r.name, r.price, r.description, r.created_at, 'rent' AS type
    FROM rent_confirmations rc
    JOIN rent r ON r.id = rc.rent_id
    WHERE rc.performer_id = ? AND rc.confirmed = true AND r.status = 'active'
    UNION ALL
    SELECT a.id, a.name, a.price, a.description, a.created_at, 'ad' AS type
    FROM ad_confirmations ac
    JOIN ad a ON a.id = ac.ad_id
    WHERE ac.performer_id = ? AND ac.confirmed = true AND a.status = 'active'
    UNION ALL
    SELECT wa.id, wa.name, wa.price, wa.description, wa.created_at, 'work_ad' AS type
    FROM work_ad_confirmations wac
    JOIN work_ad wa ON wa.id = wac.work_ad_id
    WHERE wac.performer_id = ? AND wac.confirmed = true AND wa.status = 'active'
    UNION ALL
    SELECT ra.id, ra.name, ra.price, ra.description, ra.created_at, 'rent_ad' AS type
    FROM rent_ad_confirmations rac
    JOIN rent_ad ra ON ra.id = rac.rent_ad_id
    WHERE rac.performer_id = ? AND rac.confirmed = true AND ra.status = 'active'
    ORDER BY created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

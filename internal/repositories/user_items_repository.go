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
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetOrderHistoryByUserID returns completed service, work, rent, ad, work_ad and rent_ad items for the user ordered by creation time.
func (r *UserItemsRepository) GetOrderHistoryByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT id, name, price, description, created_at, status, type FROM (
        SELECT id, name, price, description, created_at, status, 'service' AS type FROM service WHERE user_id = ? AND status = 'done'
        UNION
        SELECT s.id, s.name, s.price, s.description, s.created_at, s.status, 'service' AS type
        FROM service s
        JOIN service_responses sr ON sr.service_id = s.id
        WHERE sr.user_id = ? AND s.status IN ('done', 'in progress')
        UNION
        SELECT id, name, price, description, created_at, status, 'work' AS type FROM work WHERE user_id = ? AND status = 'done'
        UNION
        SELECT w.id, w.name, w.price, w.description, w.created_at, w.status, 'work' AS type
        FROM work w
        JOIN work_responses wr ON wr.work_id = w.id
        WHERE wr.user_id = ? AND w.status IN ('done', 'in progress')
        UNION
        SELECT id, name, price, description, created_at, status, 'rent' AS type FROM rent WHERE user_id = ? AND status = 'done'
        UNION
        SELECT r.id, r.name, r.price, r.description, r.created_at, r.status, 'rent' AS type
        FROM rent r
        JOIN rent_responses rr ON rr.rent_id = r.id
        WHERE rr.user_id = ? AND r.status IN ('done', 'in progress')
        UNION
        SELECT id, name, price, description, created_at, status, 'ad' AS type FROM ad WHERE user_id = ? AND status = 'done'
        UNION
        SELECT a.id, a.name, a.price, a.description, a.created_at, a.status, 'ad' AS type
        FROM ad a
        JOIN ad_responses ar ON ar.ad_id = a.id
        WHERE ar.user_id = ? AND a.status IN ('done', 'in progress')
        UNION
        SELECT id, name, price, description, created_at, status, 'work_ad' AS type FROM work_ad WHERE user_id = ? AND status = 'done'
        UNION
        SELECT wa.id, wa.name, wa.price, wa.description, wa.created_at, wa.status, 'work_ad' AS type
        FROM work_ad wa
        JOIN work_ad_responses war ON war.work_ad_id = wa.id
        WHERE war.user_id = ? AND wa.status IN ('done', 'in progress')
        UNION
        SELECT id, name, price, description, created_at, status, 'rent_ad' AS type FROM rent_ad WHERE user_id = ? AND status = 'done'
        UNION
        SELECT ra.id, ra.name, ra.price, ra.description, ra.created_at, ra.status, 'rent_ad' AS type
        FROM rent_ad ra
        JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id
        WHERE rar.user_id = ? AND ra.status IN ('done', 'in progress')
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetActiveOrdersByUserID returns all orders with status active, pending, or done where the user is the performer.
func (r *UserItemsRepository) GetActiveOrdersByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	query := `
    SELECT s.id, s.name, s.price, s.description, s.created_at, s.status, 'service' AS type
    FROM service_confirmations sc
    JOIN service s ON s.id = sc.service_id
    WHERE sc.performer_id = ? AND sc.confirmed = true AND s.status IN ('active', 'pending', 'done')
    UNION ALL
    SELECT w.id, w.name, w.price, w.description, w.created_at, w.status, 'work' AS type
    FROM work_confirmations wc
    JOIN work w ON w.id = wc.work_id
    WHERE wc.performer_id = ? AND wc.confirmed = true AND w.status IN ('active', 'pending', 'done')
    UNION ALL
    SELECT r.id, r.name, r.price, r.description, r.created_at, r.status, 'rent' AS type
    FROM rent_confirmations rc
    JOIN rent r ON r.id = rc.rent_id
    WHERE rc.performer_id = ? AND rc.confirmed = true AND r.status IN ('active', 'pending', 'done')
    UNION ALL
    SELECT a.id, a.name, a.price, a.description, a.created_at, a.status, 'ad' AS type
    FROM ad_confirmations ac
    JOIN ad a ON a.id = ac.ad_id
    WHERE ac.performer_id = ? AND ac.confirmed = true AND a.status IN ('active', 'pending', 'done')
    UNION ALL
    SELECT wa.id, wa.name, wa.price, wa.description, wa.created_at, wa.status, 'work_ad' AS type
    FROM work_ad_confirmations wac
    JOIN work_ad wa ON wa.id = wac.work_ad_id
    WHERE wac.performer_id = ? AND wac.confirmed = true AND wa.status IN ('active', 'pending', 'done')
    UNION ALL
    SELECT ra.id, ra.name, ra.price, ra.description, ra.created_at, ra.status, 'rent_ad' AS type
    FROM rent_ad_confirmations rac
    JOIN rent_ad ra ON ra.id = rac.rent_ad_id
    WHERE rac.performer_id = ? AND rac.confirmed = true AND ra.status IN ('active', 'pending', 'done')
    ORDER BY created_at DESC`

	rows, err := r.DB.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.UserItem
	for rows.Next() {
		var item models.UserItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.CreatedAt, &item.Status, &item.Type); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

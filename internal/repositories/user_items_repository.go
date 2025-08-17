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

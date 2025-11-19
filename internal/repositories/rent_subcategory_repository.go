package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type RentSubcategoryRepository struct {
	DB *sql.DB
}

func (r *RentSubcategoryRepository) CreateSubcategory(ctx context.Context, s models.RentSubcategory) (models.RentSubcategory, error) {
	query := `INSERT INTO rent_subcategories (category_id, name, name_kz) VALUES (?, ?, ?)`
	result, err := r.DB.ExecContext(ctx, query, s.CategoryID, s.Name, s.NameKz)
	if err != nil {
		return models.RentSubcategory{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RentSubcategory{}, err
	}
	s.ID = int(id)
	return s, nil
}

func (r *RentSubcategoryRepository) GetAllSubcategories(ctx context.Context) ([]models.RentSubcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, name_kz, created_at, updated_at FROM rent_subcategories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.RentSubcategory
	for rows.Next() {
		var s models.RentSubcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *RentSubcategoryRepository) GetByCategoryID(ctx context.Context, categoryID int) ([]models.RentSubcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, name_kz, created_at, updated_at FROM rent_subcategories WHERE category_id = ?`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.RentSubcategory
	for rows.Next() {
		var s models.RentSubcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *RentSubcategoryRepository) GetSubcategoryByID(ctx context.Context, id int) (models.RentSubcategory, error) {
	query := `
        SELECT id, category_id, name, name_kz, created_at, updated_at
        FROM rent_subcategories
        WHERE id = ?
    `
	var s models.RentSubcategory
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return models.RentSubcategory{}, models.ErrSubcategoryNotFound
	}
	if err != nil {
		return models.RentSubcategory{}, err
	}
	return s, nil
}

func (r *RentSubcategoryRepository) UpdateSubcategoryByID(ctx context.Context, sub models.RentSubcategory) (models.RentSubcategory, error) {
	query := `
        UPDATE rent_subcategories
        SET name = ?, name_kz = ?, category_id = ?, updated_at = ?
        WHERE id = ?
    `
	now := time.Now()
	sub.UpdatedAt = &now
	result, err := r.DB.ExecContext(ctx, query, sub.Name, sub.NameKz, sub.CategoryID, sub.UpdatedAt, sub.ID)
	if err != nil {
		return models.RentSubcategory{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.RentSubcategory{}, err
	}
	if rowsAffected == 0 {
		return models.RentSubcategory{}, models.ErrSubcategoryNotFound
	}
	return sub, nil
}

func (r *RentSubcategoryRepository) DeleteSubcategoryByID(ctx context.Context, id int) error {
	query := `DELETE FROM rent_subcategories WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return models.ErrSubcategoryNotFound
	}
	return nil
}

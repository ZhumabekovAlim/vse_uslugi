package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

type WorkSubcategoryRepository struct {
	DB *sql.DB
}

func (r *WorkSubcategoryRepository) CreateSubcategory(ctx context.Context, s models.WorkSubcategory) (models.WorkSubcategory, error) {
	query := `INSERT INTO work_subcategories (category_id, name, name_kz) VALUES (?, ?, ?)`
	result, err := r.DB.ExecContext(ctx, query, s.CategoryID, s.Name, s.NameKz)
	if err != nil {
		return models.WorkSubcategory{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.WorkSubcategory{}, err
	}
	s.ID = int(id)
	return s, nil
}

func (r *WorkSubcategoryRepository) GetAllSubcategories(ctx context.Context) ([]models.WorkSubcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, name_kz, created_at, updated_at FROM work_subcategories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.WorkSubcategory
	for rows.Next() {
		var s models.WorkSubcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *WorkSubcategoryRepository) GetByCategoryID(ctx context.Context, categoryID int) ([]models.WorkSubcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, name_kz, created_at, updated_at FROM work_subcategories WHERE category_id = ?`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.WorkSubcategory
	for rows.Next() {
		var s models.WorkSubcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *WorkSubcategoryRepository) GetSubcategoryByID(ctx context.Context, id int) (models.WorkSubcategory, error) {
	query := `
        SELECT id, category_id, name, name_kz, created_at, updated_at
        FROM work_subcategories
        WHERE id = ?
    `
	var s models.WorkSubcategory
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.CategoryID, &s.Name, &s.NameKz, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return models.WorkSubcategory{}, models.ErrSubcategoryNotFound
	}
	if err != nil {
		return models.WorkSubcategory{}, err
	}
	return s, nil
}

func (r *WorkSubcategoryRepository) UpdateSubcategoryByID(ctx context.Context, sub models.WorkSubcategory) (models.WorkSubcategory, error) {
	query := `
        UPDATE work_subcategories
        SET name = ?, name_kz = ?, category_id = ?, updated_at = ?
        WHERE id = ?
    `
	now := time.Now()
	sub.UpdatedAt = &now
	result, err := r.DB.ExecContext(ctx, query, sub.Name, sub.NameKz, sub.CategoryID, sub.UpdatedAt, sub.ID)
	if err != nil {
		return models.WorkSubcategory{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.WorkSubcategory{}, err
	}
	if rowsAffected == 0 {
		return models.WorkSubcategory{}, models.ErrSubcategoryNotFound
	}
	return sub, nil
}

func (r *WorkSubcategoryRepository) DeleteSubcategoryByID(ctx context.Context, id int) error {
	query := `DELETE FROM work_subcategories WHERE id = ?`
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

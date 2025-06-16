package repositories

import (
	"context"
	"database/sql"
	"errors"
	"naimuBack/internal/models"
	"time"
)

var ErrSubcategoryNotFound = errors.New("subcategory not found")

type SubcategoryRepository struct {
	DB *sql.DB
}

func (r *SubcategoryRepository) CreateSubcategory(ctx context.Context, s models.Subcategory) (models.Subcategory, error) {
	query := `INSERT INTO subcategories (category_id, name) VALUES (?, ?)`
	result, err := r.DB.ExecContext(ctx, query, s.CategoryID, s.Name)
	if err != nil {
		return models.Subcategory{}, err
	}
	id, _ := result.LastInsertId()
	s.ID = int(id)
	return s, nil
}

func (r *SubcategoryRepository) GetAllSubcategories(ctx context.Context) ([]models.Subcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, created_at, updated_at FROM subcategories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subcategory
	for rows.Next() {
		var s models.Subcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *SubcategoryRepository) GetByCategoryID(ctx context.Context, categoryID int) ([]models.Subcategory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, category_id, name, created_at, updated_at FROM subcategories WHERE category_id = ?`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subcategory
	for rows.Next() {
		var s models.Subcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *SubcategoryRepository) GetSubcategoryByID(ctx context.Context, id int) (models.Subcategory, error) {
	query := `
        SELECT id, category_id, name, created_at, updated_at
        FROM subcategories
        WHERE id = ?
    `
	var s models.Subcategory
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.CategoryID, &s.Name, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.Subcategory{}, ErrSubcategoryNotFound
	}
	if err != nil {
		return models.Subcategory{}, err
	}
	return s, nil
}

func (r *SubcategoryRepository) UpdateSubcategoryByID(ctx context.Context, sub models.Subcategory) (models.Subcategory, error) {
	query := `
		UPDATE subcategories 
		SET name = ?, category_id = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	sub.UpdatedAt = &now

	result, err := r.DB.ExecContext(ctx, query, sub.Name, sub.CategoryID, sub.UpdatedAt, sub.ID)
	if err != nil {
		return models.Subcategory{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Subcategory{}, err
	}
	if rowsAffected == 0 {
		return models.Subcategory{}, errors.New("subcategory not found")
	}

	return sub, nil
}

func (r *SubcategoryRepository) DeleteSubcategoryByID(ctx context.Context, id int) error {
	query := `DELETE FROM subcategories WHERE id = ?`

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

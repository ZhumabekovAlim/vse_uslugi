package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

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

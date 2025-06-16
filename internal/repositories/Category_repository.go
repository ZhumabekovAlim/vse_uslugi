package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
	"time"
)

var (
	ErrCategoryNotFound = models.ErrCategoryNotFound
)

type CategoryRepository struct {
	DB *sql.DB
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Category{}, err
	}

	insertCategory := `
		INSERT INTO categories (name, image_path, min_price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertCategory,
		category.Name, category.ImagePath, category.MinPrice, category.CreatedAt, category.UpdatedAt,
	).Scan(&category.ID)
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}

	insertLink := `
		INSERT INTO category_subcategory (category_id, subcategory_id) VALUES (?, ?)
	`
	for _, subID := range category.SubcategoryIDs {
		if _, err := tx.ExecContext(ctx, insertLink, category.ID, subID); err != nil {
			tx.Rollback()
			return models.Category{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id int) (models.Category, error) {
	var category models.Category
	query := `
        SELECT id, name, image_path, subcategories, min_price, created_at, updated_at
        FROM categories
        WHERE id = ?
    `
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.ImagePath,
		&category.MinPrice, &category.CreatedAt, &category.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.Category{}, ErrCategoryNotFound
	}
	if err != nil {
		return models.Category{}, err
	}
	return category, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	query := `
        UPDATE categories
        SET name = ?, image_path = ?, subcategories = ?, min_price = ?, updated_at = ?
        WHERE id = ?
    `
	updatedAt := time.Now()
	category.UpdatedAt = updatedAt
	result, err := r.DB.ExecContext(ctx, query,
		category.Name, category.ImagePath, category.MinPrice,
		category.UpdatedAt, category.ID,
	)
	if err != nil {
		return models.Category{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Category{}, err
	}
	if rowsAffected == 0 {
		return models.Category{}, ErrCategoryNotFound
	}
	return r.GetCategoryByID(ctx, category.ID)
}

func (r *CategoryRepository) DeleteCategory(ctx context.Context, id int) error {
	query := `DELETE FROM categories WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *CategoryRepository) GetAllCategories(ctx context.Context) ([]models.Category, error) {
	query := `
        SELECT id, name, image_path, min_price, created_at, updated_at
        FROM categories
    `
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		err := rows.Scan(
			&c.ID, &c.Name, &c.ImagePath,
			&c.MinPrice, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Загружаем подкатегории по связующей таблице category_subcategory
		subQuery := `
			SELECT s.id, s.category_id, s.name, s.created_at, s.updated_at
			FROM subcategories s
			JOIN category_subcategory cs ON cs.subcategory_id = s.id
			WHERE cs.category_id = ?
		`
		subRows, err := r.DB.QueryContext(ctx, subQuery, c.ID)
		if err != nil {
			return nil, err
		}

		var subcats []models.Subcategory
		for subRows.Next() {
			var sub models.Subcategory
			if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				subRows.Close()
				return nil, err
			}
			subcats = append(subcats, sub)
		}
		subRows.Close()

		c.Subcategories = subcats
		categories = append(categories, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

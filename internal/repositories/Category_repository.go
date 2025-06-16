package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"naimuBack/internal/models"
	"strings"
	"time"
)

var (
	ErrCategoryNotFound = models.ErrCategoryNotFound
)

type CategoryRepository struct {
	DB *sql.DB
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, category models.Category, subcategoryIDs []int) (models.Category, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Category{}, err
	}

	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	// 1. Вставка категории
	query := `
		INSERT INTO categories (name, image_path, min_price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query, category.Name, category.ImagePath, category.MinPrice, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}
	categoryID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}
	category.ID = int(categoryID)

	// 2. Связка с subcategory_id в связующей таблице
	linkQuery := `INSERT INTO category_subcategory (category_id, subcategory_id) VALUES `
	vals := []interface{}{}
	for _, sid := range subcategoryIDs {
		linkQuery += `(?, ?),`
		vals = append(vals, category.ID, sid)
	}
	linkQuery = strings.TrimSuffix(linkQuery, ",")
	_, err = tx.ExecContext(ctx, linkQuery, vals...)
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}

	// 3. Получаем названия subcategory по их id
	if len(subcategoryIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(subcategoryIDs)), ",")
		args := make([]interface{}, len(subcategoryIDs))
		for i, id := range subcategoryIDs {
			args[i] = id
		}
		subQuery := fmt.Sprintf(`
			SELECT id, category_id, name, created_at, updated_at 
			FROM subcategories 
			WHERE id IN (%s)
		`, placeholders)

		rows, err := tx.QueryContext(ctx, subQuery, args...)
		if err != nil {
			tx.Rollback()
			return models.Category{}, err
		}
		defer rows.Close()

		for rows.Next() {
			var sub models.Subcategory
			if err := rows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				tx.Rollback()
				return models.Category{}, err
			}
			category.Subcategories = append(category.Subcategories, sub)
		}
		if err := rows.Err(); err != nil {
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
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.MinPrice, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}

		// --- Загрузка подкатегорий для данной категории ---
		subRows, err := r.DB.QueryContext(ctx, `
			SELECT id, category_id, name, created_at, updated_at
			FROM subcategories
			WHERE category_id = ?
		`, c.ID)
		if err != nil {
			return nil, err
		}

		var subcategories []models.Subcategory
		for subRows.Next() {
			var s models.Subcategory
			if err := subRows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
				subRows.Close() // безопасно закрываем перед возвратом ошибки
				return nil, err
			}
			subcategories = append(subcategories, s)
		}
		subRows.Close()

		c.Subcategories = subcategories
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

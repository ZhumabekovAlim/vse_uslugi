package repositories

import (
	"context"
	"database/sql"
	_ "fmt"
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

func (r *CategoryRepository) CreateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Category{}, err
	}

	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	// Вставка категории
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

	if err := tx.Commit(); err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id int) (models.Category, error) {
	var category models.Category

	query := `
		SELECT id, name, image_path, min_price, created_at, updated_at
		FROM categories
		WHERE id = ?
	`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.ImagePath,
		&category.MinPrice,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Category{}, ErrCategoryNotFound
		}
		return models.Category{}, err
	}

	// Подгружаем связанные подкатегории
	subRows, err := r.DB.QueryContext(ctx, `
		SELECT id, category_id, name, created_at, updated_at
		FROM subcategories
		WHERE category_id = ?
	`, category.ID)
	if err != nil {
		return models.Category{}, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var sub models.Subcategory
		if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return models.Category{}, err
		}
		category.Subcategories = append(category.Subcategories, sub)
	}

	return category, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, category models.Category, subcategoryIDs []int) (models.Category, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Category{}, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Обновляем основную категорию
	query := `
		UPDATE categories
		SET name = ?, image_path = ?, min_price = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, query,
		category.Name, category.ImagePath, category.MinPrice, time.Now(), category.ID,
	)
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}

	// Удаляем старые связи с подкатегориями
	_, err = tx.ExecContext(ctx, `DELETE FROM category_subcategory WHERE category_id = ?`, category.ID)
	if err != nil {
		tx.Rollback()
		return models.Category{}, err
	}

	// Вставляем новые связи
	for _, subID := range subcategoryIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO category_subcategory (category_id, subcategory_id) VALUES (?, ?)`, category.ID, subID)
		if err != nil {
			tx.Rollback()
			return models.Category{}, err
		}
	}

	// Получаем данные подкатегорий
	if len(subcategoryIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(subcategoryIDs)), ",")
		args := make([]interface{}, len(subcategoryIDs))
		for i, id := range subcategoryIDs {
			args[i] = id
		}

		subRows, err := tx.QueryContext(ctx, `
			SELECT id, category_id, name, created_at, updated_at
			FROM subcategories
			WHERE id IN (`+placeholders+`)`, args...,
		)
		if err != nil {
			tx.Rollback()
			return models.Category{}, err
		}
		defer subRows.Close()

		for subRows.Next() {
			var sub models.Subcategory
			if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				tx.Rollback()
				return models.Category{}, err
			}
			category.Subcategories = append(category.Subcategories, sub)
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Category{}, err
	}

	return category, nil
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

		// Загружаем субкатегории через связующую таблицу
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

		for subRows.Next() {
			var sub models.Subcategory
			if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				subRows.Close()
				return nil, err
			}
			c.Subcategories = append(c.Subcategories, sub)
		}
		subRows.Close()

		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

package repositories

import (
	"context"
	"database/sql"
	_ "fmt"
	"naimuBack/internal/models"
	"time"
)

var (
	ErrRentCategoryNotFound = models.ErrRentCategoryNotFound
)

type RentCategoryRepository struct {
	DB *sql.DB
}

func (r *RentCategoryRepository) CreateCategory(ctx context.Context, category models.RentCategory) (models.RentCategory, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.RentCategory{}, err
	}

	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	// Вставка категории
	query := `
		INSERT INTO rent_categories (name, image_path, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query, category.Name, category.ImagePath, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return models.RentCategory{}, err
	}

	categoryID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return models.RentCategory{}, err
	}

	category.ID = int(categoryID)
	category.MinPrice = 0
	if err := tx.Commit(); err != nil {
		return models.RentCategory{}, err
	}

	return category, nil
}

func (r *RentCategoryRepository) UpdateCategory(ctx context.Context, category models.RentCategory) (models.RentCategory, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.RentCategory{}, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Обновляем имя и путь к изображению
	query := `
		UPDATE rent_categories
		SET name = ?, image_path = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, query, category.Name, category.ImagePath, time.Now(), category.ID)
	if err != nil {
		tx.Rollback()
		return models.RentCategory{}, err
	}

	// Получаем обратно обновлённые данные
	row := tx.QueryRowContext(ctx, `
		SELECT id, name, image_path, created_at, updated_at
		FROM rent_categories
		WHERE id = ?
	`, category.ID)

	var updated models.RentCategory
	err = row.Scan(
		&updated.ID,
		&updated.Name,
		&updated.ImagePath,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		tx.Rollback()
		return models.RentCategory{}, err
	}

	// Подтягиваем min_price
	priceQuery := `SELECT MIN(price) FROM rent WHERE category_id = ?`
	_ = tx.QueryRowContext(ctx, priceQuery, updated.ID).Scan(&updated.MinPrice)

	// Подтягиваем rent_subcategories
	subQuery := `
		SELECT id, category_id, name, created_at, updated_at
		FROM rent_subcategories
		WHERE category_id = ?
	`
	subRows, err := tx.QueryContext(ctx, subQuery, updated.ID)
	if err != nil {
		tx.Rollback()
		return models.RentCategory{}, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var sub models.RentSubcategory
		if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			tx.Rollback()
			return models.RentCategory{}, err
		}
		updated.Subcategories = append(updated.Subcategories, sub)
	}

	if err := tx.Commit(); err != nil {
		return models.RentCategory{}, err
	}

	return updated, nil
}

func (r *RentCategoryRepository) DeleteCategory(ctx context.Context, id int) error {
	query := `DELETE FROM rent_categories WHERE id = ?`
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

func (r *RentCategoryRepository) GetAllCategories(ctx context.Context) ([]models.RentCategory, error) {
	var rent_categories []models.RentCategory

	query := `SELECT id, name, image_path, created_at, updated_at FROM rent_categories`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category models.RentCategory
		err := rows.Scan(&category.ID, &category.Name, &category.ImagePath, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, err
		}

		priceQuery := `SELECT MIN(price) FROM rent WHERE category_id = ?`
		_ = r.DB.QueryRowContext(ctx, priceQuery, category.ID).Scan(&category.MinPrice)

		// Получаем подкатегории для каждой категории
		subQuery := `
			SELECT id, category_id, name, created_at, updated_at
			FROM rent_subcategories
			WHERE category_id = ?
		`
		subRows, err := r.DB.QueryContext(ctx, subQuery, category.ID)
		if err != nil {
			return nil, err
		}

		for subRows.Next() {
			var sub models.RentSubcategory
			if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				return nil, err
			}
			category.Subcategories = append(category.Subcategories, sub)
		}
		subRows.Close()

		rent_categories = append(rent_categories, category)
	}

	return rent_categories, nil
}

func (r *RentCategoryRepository) GetCategoryByID(ctx context.Context, id int) (models.RentCategory, error) {
	var category models.RentCategory

	// Получаем саму категорию
	query := `
		SELECT id, name, image_path, created_at, updated_at
		FROM rent_categories
		WHERE id = ?
	`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.ImagePath,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	priceQuery := `SELECT MIN(price) FROM rent WHERE category_id = ?`
	_ = r.DB.QueryRowContext(ctx, priceQuery, category.ID).Scan(&category.MinPrice)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.RentCategory{}, nil
		}
		return models.RentCategory{}, err
	}

	// Загружаем связанные субкатегории через category_subcategory
	subQuery := `
			SELECT id, category_id, name, created_at, updated_at
			FROM rent_subcategories
			WHERE category_id = ?
		`
	rows, err := r.DB.QueryContext(ctx, subQuery, id)
	if err != nil {
		return category, err
	}
	defer rows.Close()

	for rows.Next() {
		var sub models.RentSubcategory
		if err := rows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return category, err
		}
		category.Subcategories = append(category.Subcategories, sub)
	}

	if err := rows.Err(); err != nil {
		return category, err
	}

	return category, nil
}

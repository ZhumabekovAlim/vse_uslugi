package repositories

import (
	"context"
	"database/sql"
	_ "fmt"
	"naimuBack/internal/models"
	"time"
)

type WorkCategoryRepository struct {
	DB *sql.DB
}

func (r *WorkCategoryRepository) CreateCategory(ctx context.Context, category models.WorkCategory) (models.WorkCategory, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.WorkCategory{}, err
	}

	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	// Вставка категории
	query := `
		INSERT INTO work_categories (name, image_path, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query, category.Name, category.ImagePath, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return models.WorkCategory{}, err
	}

	categoryID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return models.WorkCategory{}, err
	}

	category.ID = int(categoryID)
	category.MinPrice = 0
	if err := tx.Commit(); err != nil {
		return models.WorkCategory{}, err
	}

	return category, nil
}

func (r *WorkCategoryRepository) UpdateCategory(ctx context.Context, category models.WorkCategory) (models.WorkCategory, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.WorkCategory{}, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Обновляем имя и путь к изображению
	query := `
		UPDATE work_categories
		SET name = ?, image_path = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, query, category.Name, category.ImagePath, time.Now(), category.ID)
	if err != nil {
		tx.Rollback()
		return models.WorkCategory{}, err
	}

	// Получаем обратно обновлённые данные
	row := tx.QueryRowContext(ctx, `
                SELECT id, name, image_path, created_at, updated_at
                FROM work_categories
                WHERE id = ?
        `, category.ID)

	var updated models.WorkCategory
	var imagePath sql.NullString
	err = row.Scan(
		&updated.ID,
		&updated.Name,
		&imagePath,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		tx.Rollback()
		return models.WorkCategory{}, err
	}

	if imagePath.Valid {
		updated.ImagePath = imagePath.String
	}

	// Подтягиваем min_price
	priceQuery := `SELECT MIN(price) FROM work WHERE category_id = ?`
	_ = tx.QueryRowContext(ctx, priceQuery, updated.ID).Scan(&updated.MinPrice)

	// Подтягиваем work_subcategories
	subQuery := `
		SELECT id, category_id, name, created_at, updated_at
		FROM work_subcategories
		WHERE category_id = ?
	`
	subRows, err := tx.QueryContext(ctx, subQuery, updated.ID)
	if err != nil {
		tx.Rollback()
		return models.WorkCategory{}, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var sub models.WorkSubcategory
		if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			tx.Rollback()
			return models.WorkCategory{}, err
		}
		updated.Subcategories = append(updated.Subcategories, sub)
	}

	if err := tx.Commit(); err != nil {
		return models.WorkCategory{}, err
	}

	return updated, nil
}

func (r *WorkCategoryRepository) DeleteCategory(ctx context.Context, id int) error {
	query := `DELETE FROM work_categories WHERE id = ?`
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

func (r *WorkCategoryRepository) GetAllCategories(ctx context.Context) ([]models.WorkCategory, error) {
	var work_categories []models.WorkCategory

	query := `SELECT id, name, image_path, created_at, updated_at FROM work_categories`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category models.WorkCategory
		var imagePath sql.NullString
		err := rows.Scan(&category.ID, &category.Name, &imagePath, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if imagePath.Valid {
			category.ImagePath = imagePath.String
		}

		priceQuery := `SELECT MIN(price) FROM work WHERE category_id = ?`
		_ = r.DB.QueryRowContext(ctx, priceQuery, category.ID).Scan(&category.MinPrice)

		// Получаем подкатегории для каждой категории
		subQuery := `
			SELECT id, category_id, name, created_at, updated_at
			FROM work_subcategories
			WHERE category_id = ?
		`
		subRows, err := r.DB.QueryContext(ctx, subQuery, category.ID)
		if err != nil {
			return nil, err
		}

		for subRows.Next() {
			var sub models.WorkSubcategory
			if err := subRows.Scan(&sub.ID, &sub.CategoryID, &sub.Name, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
				return nil, err
			}
			category.Subcategories = append(category.Subcategories, sub)
		}
		subRows.Close()

		work_categories = append(work_categories, category)
	}

	return work_categories, nil
}

func (r *WorkCategoryRepository) GetCategoryByID(ctx context.Context, id int) (models.WorkCategory, error) {
	var category models.WorkCategory

	// Получаем саму категорию
	query := `
		SELECT id, name, image_path, created_at, updated_at
		FROM work_categories
		WHERE id = ?
	`
	var imagePath sql.NullString
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&imagePath,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	priceQuery := `SELECT MIN(price) FROM work WHERE category_id = ?`
	_ = r.DB.QueryRowContext(ctx, priceQuery, category.ID).Scan(&category.MinPrice)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.WorkCategory{}, nil
		}
		return models.WorkCategory{}, err
	}

	if imagePath.Valid {
		category.ImagePath = imagePath.String
	}

	// Загружаем связанные субкатегории через category_subcategory
	subQuery := `
			SELECT id, category_id, name, created_at, updated_at
			FROM work_subcategories
			WHERE category_id = ?
		`
	rows, err := r.DB.QueryContext(ctx, subQuery, id)
	if err != nil {
		return category, err
	}
	defer rows.Close()

	for rows.Next() {
		var sub models.WorkSubcategory
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

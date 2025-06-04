package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type CityRepository struct {
	DB *sql.DB
}

func (r *CityRepository) CreateCity(ctx context.Context, city models.City) (models.City, error) {
	query := `INSERT INTO cities (name, created_at, updated_at) VALUES (?, NOW(), NOW())`
	res, err := r.DB.ExecContext(ctx, query, city.Name)
	if err != nil {
		return models.City{}, err
	}
	id, _ := res.LastInsertId()
	city.ID = int(id)
	return city, nil
}

func (r *CityRepository) GetCities(ctx context.Context) ([]models.City, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, created_at, updated_at FROM cities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []models.City
	for rows.Next() {
		var city models.City
		if err := rows.Scan(&city.ID, &city.Name, &city.CreatedAt, &city.UpdatedAt); err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}
	return cities, nil
}

func (r *CityRepository) GetCityByID(ctx context.Context, id int) (models.City, error) {
	var city models.City
	query := `SELECT id, name, created_at, updated_at FROM cities WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&city.ID, &city.Name, &city.CreatedAt, &city.UpdatedAt)
	return city, err
}

func (r *CityRepository) UpdateCity(ctx context.Context, city models.City) (models.City, error) {
	query := `UPDATE cities SET name = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, city.Name, city.ID)
	return city, err
}

func (r *CityRepository) DeleteCity(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM cities WHERE id = ?`, id)
	return err
}

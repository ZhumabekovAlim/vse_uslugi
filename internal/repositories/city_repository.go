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
	query := `INSERT INTO cities (name, type, parent_id, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`
	res, err := r.DB.ExecContext(ctx, query, city.Name, city.Type, city.ParentID)
	if err != nil {
		return models.City{}, err
	}
	id, _ := res.LastInsertId()
	city.ID = int(id)
	return city, nil
}

func (r *CityRepository) GetCities(ctx context.Context) ([]models.City, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, type, parent_id, created_at, updated_at FROM cities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []models.City
	for rows.Next() {
		var city models.City
		var parentID sql.NullInt64
		if err := rows.Scan(&city.ID, &city.Name, &city.Type, &parentID, &city.CreatedAt, &city.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			city.ParentID = &pid
		}
		all = append(all, city)
	}

	// Build hierarchy
	cityMap := make(map[int]*models.City)
	for i := range all {
		cityMap[all[i].ID] = &all[i]
	}

	for i := range all {
		c := &all[i]
		if c.ParentID != nil {
			if parent, ok := cityMap[*c.ParentID]; ok {
				parent.Cities = append(parent.Cities, *c)
			}
		}
	}

	var roots []models.City
	for i := range all {
		if all[i].ParentID == nil {
			roots = append(roots, all[i])
		}
	}

	return roots, nil
}

func (r *CityRepository) GetCityByID(ctx context.Context, id int) (models.City, error) {
	var city models.City
	var parentID sql.NullInt64
	query := `SELECT id, name, type, parent_id, created_at, updated_at FROM cities WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&city.ID, &city.Name, &city.Type, &parentID, &city.CreatedAt, &city.UpdatedAt)
	if parentID.Valid {
		pid := int(parentID.Int64)
		city.ParentID = &pid
	}
	if err != nil {
		return city, err
	}

	if city.Type == "region" {
		rows, err := r.DB.QueryContext(ctx, `SELECT id, name, type, parent_id, created_at, updated_at FROM cities WHERE parent_id = ?`, city.ID)
		if err != nil {
			return city, err
		}
		defer rows.Close()

		for rows.Next() {
			var c models.City
			var pID sql.NullInt64
			if err := rows.Scan(&c.ID, &c.Name, &c.Type, &pID, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return city, err
			}
			if pID.Valid {
				pid := int(pID.Int64)
				c.ParentID = &pid
			}
			city.Cities = append(city.Cities, c)
		}
	}

	return city, nil
}

func (r *CityRepository) UpdateCity(ctx context.Context, city models.City) (models.City, error) {
	query := `UPDATE cities SET name = ?, type = ?, parent_id = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, city.Name, city.Type, city.ParentID, city.ID)
	return city, err
}

func (r *CityRepository) DeleteCity(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM cities WHERE id = ?`, id)
	return err
}

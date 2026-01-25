package dispatch

import (
	"context"
	"database/sql"
	"errors"
)

type DriversRepo struct {
	DB *sql.DB
}

func NewDriversRepo(db *sql.DB) *DriversRepo {
	return &DriversRepo{DB: db}
}

func (r *DriversRepo) Exists(ctx context.Context, driverID int64) (bool, error) {
	var x int
	err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM drivers WHERE id = ? LIMIT 1`, driverID).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

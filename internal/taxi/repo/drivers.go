package repo

import (
	"context"
	"database/sql"
	"time"
)

// Driver represents a driver profile in the taxi module.
type Driver struct {
	ID            int64
	UserID        int64
	Status        string
	CarModel      sql.NullString
	CarColor      sql.NullString
	CarNumber     string
	TechPassport  string
	CarPhotoFront string
	CarPhotoBack  string
	CarPhotoLeft  string
	CarPhotoRight string
	DriverPhoto   string
	Phone         string
	IIN           string
	IDCardFront   string
	IDCardBack    string
	Rating        float64
	UpdatedAt     time.Time
}

// DriversRepo provides CRUD operations for drivers.
type DriversRepo struct {
	db *sql.DB
}

// NewDriversRepo constructs a DriversRepo.
func NewDriversRepo(db *sql.DB) *DriversRepo {
	return &DriversRepo{db: db}
}

// Create inserts a new driver record and returns its id.
func (r *DriversRepo) Create(ctx context.Context, d Driver) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO drivers (
        user_id, status, car_model, car_color, car_number, tech_passport,
        car_photo_front, car_photo_back, car_photo_left, car_photo_right,
        driver_photo, phone, iin, id_card_front, id_card_back
    ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.UserID, d.Status, d.CarModel, d.CarColor, d.CarNumber, d.TechPassport,
		d.CarPhotoFront, d.CarPhotoBack, d.CarPhotoLeft, d.CarPhotoRight,
		d.DriverPhoto, d.Phone, d.IIN, d.IDCardFront, d.IDCardBack,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Get fetches a driver by id.
func (r *DriversRepo) Get(ctx context.Context, id int64) (Driver, error) {
	var d Driver
	row := r.db.QueryRowContext(ctx, `SELECT
        id, user_id, status, car_model, car_color, car_number, tech_passport,
        car_photo_front, car_photo_back, car_photo_left, car_photo_right,
        driver_photo, phone, iin, id_card_front, id_card_back, rating, updated_at
    FROM drivers WHERE id = ?`, id)
	err := row.Scan(&d.ID, &d.UserID, &d.Status, &d.CarModel, &d.CarColor, &d.CarNumber, &d.TechPassport,
		&d.CarPhotoFront, &d.CarPhotoBack, &d.CarPhotoLeft, &d.CarPhotoRight,
		&d.DriverPhoto, &d.Phone, &d.IIN, &d.IDCardFront, &d.IDCardBack, &d.Rating, &d.UpdatedAt)
	if err != nil {
		return Driver{}, err
	}
	return d, nil
}

// List returns drivers with limit and offset.
func (r *DriversRepo) List(ctx context.Context, limit, offset int) ([]Driver, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT
        id, user_id, status, car_model, car_color, car_number, tech_passport,
        car_photo_front, car_photo_back, car_photo_left, car_photo_right,
        driver_photo, phone, iin, id_card_front, id_card_back, rating, updated_at
    FROM drivers ORDER BY id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []Driver
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.UserID, &d.Status, &d.CarModel, &d.CarColor, &d.CarNumber, &d.TechPassport,
			&d.CarPhotoFront, &d.CarPhotoBack, &d.CarPhotoLeft, &d.CarPhotoRight,
			&d.DriverPhoto, &d.Phone, &d.IIN, &d.IDCardFront, &d.IDCardBack, &d.Rating, &d.UpdatedAt); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return drivers, nil
}

// Update overwrites existing driver data.
func (r *DriversRepo) Update(ctx context.Context, d Driver) error {
	res, err := r.db.ExecContext(ctx, `UPDATE drivers SET
        user_id = ?, status = ?, car_model = ?, car_color = ?, car_number = ?, tech_passport = ?,
        car_photo_front = ?, car_photo_back = ?, car_photo_left = ?, car_photo_right = ?,
        driver_photo = ?, phone = ?, iin = ?, id_card_front = ?, id_card_back = ?
    WHERE id = ?`,
		d.UserID, d.Status, d.CarModel, d.CarColor, d.CarNumber, d.TechPassport,
		d.CarPhotoFront, d.CarPhotoBack, d.CarPhotoLeft, d.CarPhotoRight,
		d.DriverPhoto, d.Phone, d.IIN, d.IDCardFront, d.IDCardBack, d.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a driver by id.
func (r *DriversRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM drivers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

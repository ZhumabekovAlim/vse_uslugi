package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Driver represents a driver profile in the taxi module.
type Driver struct {
	ID             int64
	UserID         int64
	Name           string
	Surname        string
	Middlename     sql.NullString
	Status         string
	ApprovalStatus string
	IsBanned       bool
	CarModel       sql.NullString
	CarColor       sql.NullString
	CarNumber      string
	TechPassport   string
	CarPhotoFront  string
	CarPhotoBack   string
	CarPhotoLeft   string
	CarPhotoRight  string
	DriverPhoto    string
	Phone          string
	IIN            string
	IDCardFront    string
	IDCardBack     string
	Rating         float64
	Balance        int
	UpdatedAt      time.Time
}

// DriversRepo provides CRUD operations for drivers.
type DriversRepo struct {
	db *sql.DB
}

// ErrInsufficientBalance is returned when balance adjustments would drop below zero.
var ErrInsufficientBalance = errors.New("insufficient balance")

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
        d.id, d.user_id, d.status, d.approval_status, d.is_banned, d.car_model, d.car_color, d.car_number, d.tech_passport,
        d.car_photo_front, d.car_photo_back, d.car_photo_left, d.car_photo_right,
        d.driver_photo, d.phone, d.iin, d.id_card_front, d.id_card_back, d.rating, d.balance, d.updated_at,
        u.name, u.surname, COALESCE(u.middlename, '')
    FROM drivers d
    JOIN users u ON u.id = d.user_id
    WHERE d.id = ?`, id)
	err := row.Scan(&d.ID, &d.UserID, &d.Status, &d.ApprovalStatus, &d.IsBanned, &d.CarModel, &d.CarColor, &d.CarNumber, &d.TechPassport,
		&d.CarPhotoFront, &d.CarPhotoBack, &d.CarPhotoLeft, &d.CarPhotoRight,
		&d.DriverPhoto, &d.Phone, &d.IIN, &d.IDCardFront, &d.IDCardBack, &d.Rating, &d.Balance, &d.UpdatedAt,
		&d.Name, &d.Surname, &d.Middlename)
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
        d.id, d.user_id, d.status, d.approval_status, d.is_banned, d.car_model, d.car_color, d.car_number, d.tech_passport,
        d.car_photo_front, d.car_photo_back, d.car_photo_left, d.car_photo_right,
        d.driver_photo, d.phone, d.iin, d.id_card_front, d.id_card_back, d.rating, d.balance, d.updated_at,
        u.name, u.surname, u.middlename
    FROM drivers d
    JOIN users u ON u.id = d.user_id
    ORDER BY d.id
    LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []Driver
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.UserID, &d.Status, &d.ApprovalStatus, &d.IsBanned, &d.CarModel, &d.CarColor, &d.CarNumber, &d.TechPassport,
			&d.CarPhotoFront, &d.CarPhotoBack, &d.CarPhotoLeft, &d.CarPhotoRight,
			&d.DriverPhoto, &d.Phone, &d.IIN, &d.IDCardFront, &d.IDCardBack, &d.Rating, &d.Balance, &d.UpdatedAt,
			&d.Name, &d.Surname, &d.Middlename); err != nil {
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

// Deposit increases driver's balance by amount and returns new balance.
func (r *DriversRepo) Deposit(ctx context.Context, driverID int64, amount int) (int, error) {
	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	return r.adjustBalance(ctx, driverID, amount, true)
}

// Withdraw decreases driver's balance by amount when sufficient funds are available.
func (r *DriversRepo) Withdraw(ctx context.Context, driverID int64, amount int) (int, error) {
	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	return r.adjustBalance(ctx, driverID, -amount, false)
}

// Charge deducts amount from balance allowing it to go negative (for commissions).
func (r *DriversRepo) Charge(ctx context.Context, driverID int64, amount int) (int, error) {
	if amount < 0 {
		return 0, errors.New("amount must be non-negative")
	}
	if amount == 0 {
		driver, err := r.Get(ctx, driverID)
		if err != nil {
			return 0, err
		}
		return driver.Balance, nil
	}
	return r.adjustBalance(ctx, driverID, -amount, true)
}

func (r *DriversRepo) adjustBalance(ctx context.Context, driverID int64, delta int, allowNegative bool) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var balance int
	if err = tx.QueryRowContext(ctx, `SELECT balance FROM drivers WHERE id = ? FOR UPDATE`, driverID).Scan(&balance); err != nil {
		return 0, err
	}
	newBalance := balance + delta
	if !allowNegative && newBalance < 0 {
		return 0, ErrInsufficientBalance
	}
	if _, err = tx.ExecContext(ctx, `UPDATE drivers SET balance = ? WHERE id = ?`, newBalance, driverID); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return newBalance, nil
}

// SetBanStatus updates driver's ban flag. When banning a driver their status is forced offline.
func (r *DriversRepo) SetBanStatus(ctx context.Context, driverID int64, banned bool) error {
	query := "UPDATE drivers SET is_banned = ?"
	args := []interface{}{banned}
	if banned {
		query += ", status = 'offline'"
	}
	query += " WHERE id = ?"
	args = append(args, driverID)

	res, err := r.db.ExecContext(ctx, query, args...)
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

// UpdateApprovalStatus sets driver's approval workflow state. Rejected drivers are forced offline.
func (r *DriversRepo) UpdateApprovalStatus(ctx context.Context, driverID int64, status string) error {
	switch status {
	case "approved", "rejected":
	default:
		return fmt.Errorf("invalid approval status: %s", status)
	}

	query := "UPDATE drivers SET approval_status = ?"
	args := []interface{}{status}
	if status != "approved" {
		query += ", status = 'offline'"
	}
	query += " WHERE id = ?"
	args = append(args, driverID)

	res, err := r.db.ExecContext(ctx, query, args...)
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

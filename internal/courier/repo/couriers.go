package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrInsufficientBalance is returned when balance adjustments would drop below zero.
var ErrInsufficientBalance = errors.New("insufficient balance")

// Courier represents a courier profile record.
type Courier struct {
	ID          int64
	UserID      int64
	FirstName   string
	LastName    string
	MiddleName  sql.NullString
	Photo       string
	IIN         string
	BirthDate   time.Time
	IDCardFront string
	IDCardBack  string
	Phone       string
	Rating      sql.NullFloat64
	Balance     int
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CouriersStats aggregates counters for couriers grouped by status.
type CouriersStats struct {
	Total   int `json:"total_couriers"`
	Pending int `json:"pending_couriers"`
	Active  int `json:"active_couriers"`
	Banned  int `json:"banned_couriers"`
}

// CouriersRepo provides persistence helpers for courier profiles.
type CouriersRepo struct {
	db *sql.DB
}

// NewCouriersRepo constructs a CouriersRepo.
func NewCouriersRepo(db *sql.DB) *CouriersRepo {
	return &CouriersRepo{db: db}
}

// Upsert creates or updates courier profile by user identifier.
func (r *CouriersRepo) Upsert(ctx context.Context, c Courier) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO couriers (
        user_id, first_name, last_name, middle_name, courier_photo, iin, date_of_birth,
        id_card_front, id_card_back, phone, rating, status
    ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
    ON DUPLICATE KEY UPDATE
        first_name = VALUES(first_name),
        last_name = VALUES(last_name),
        middle_name = VALUES(middle_name),
        courier_photo = VALUES(courier_photo),
        iin = VALUES(iin),
        date_of_birth = VALUES(date_of_birth),
        id_card_front = VALUES(id_card_front),
        id_card_back = VALUES(id_card_back),
        phone = VALUES(phone),
        rating = VALUES(rating),
        status = VALUES(status),
        updated_at = CURRENT_TIMESTAMP,
        id = LAST_INSERT_ID(id)`,
		c.UserID, c.FirstName, c.LastName, nullOrString(c.MiddleName), c.Photo, c.IIN, c.BirthDate,
		c.IDCardFront, c.IDCardBack, c.Phone, nullFloat64(c.Rating), c.Status,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Get returns courier by identifier.
func (r *CouriersRepo) Get(ctx context.Context, id int64) (Courier, error) {
	var c Courier
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, first_name, last_name, middle_name,
        courier_photo, iin, date_of_birth, id_card_front, id_card_back, phone, rating, balance, status, created_at, updated_at
        FROM couriers WHERE id = ?`, id)
	err := row.Scan(&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.MiddleName,
		&c.Photo, &c.IIN, &c.BirthDate, &c.IDCardFront, &c.IDCardBack, &c.Phone, &c.Rating, &c.Balance, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Courier{}, ErrNotFound
		}
		return Courier{}, err
	}
	return c, nil
}

// List returns couriers with pagination.
func (r *CouriersRepo) List(ctx context.Context, limit, offset int) ([]Courier, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, first_name, last_name, middle_name,
        courier_photo, iin, date_of_birth, id_card_front, id_card_back, phone, rating, balance, status, created_at, updated_at
        FROM couriers ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var couriers []Courier
	for rows.Next() {
		var c Courier
		if err := rows.Scan(&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.MiddleName,
			&c.Photo, &c.IIN, &c.BirthDate, &c.IDCardFront, &c.IDCardBack, &c.Phone, &c.Rating, &c.Balance, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		couriers = append(couriers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return couriers, nil
}

// Stats aggregates courier counts by status.
func (r *CouriersRepo) Stats(ctx context.Context) (CouriersStats, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM couriers GROUP BY status`)
	if err != nil {
		return CouriersStats{}, err
	}
	defer rows.Close()

	var stats CouriersStats
	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return CouriersStats{}, err
		}
		stats.Total += count
		switch status {
		case "pending":
			stats.Pending += count
		case "banned":
			stats.Banned += count
		default:
			stats.Active += count
		}
	}
	if err := rows.Err(); err != nil {
		return CouriersStats{}, err
	}
	return stats, nil
}

// UpdateStatus changes courier status.
func (r *CouriersRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE couriers SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DepositBalance increases courier balance and returns the new value.
func (r *CouriersRepo) DepositBalance(ctx context.Context, courierID int64, amount int) (int, error) {
	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	return r.adjustBalance(ctx, courierID, amount, true)
}

// WithdrawBalance decreases courier balance and returns the new value.
func (r *CouriersRepo) WithdrawBalance(ctx context.Context, courierID int64, amount int) (int, error) {
	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	return r.adjustBalance(ctx, courierID, -amount, false)
}

func (r *CouriersRepo) adjustBalance(ctx context.Context, courierID int64, delta int, allowNegative bool) (int, error) {
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
	if err = tx.QueryRowContext(ctx, `SELECT balance FROM couriers WHERE id = ? FOR UPDATE`, courierID).Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	newBalance := balance + delta
	if !allowNegative && newBalance < 0 {
		return 0, ErrInsufficientBalance
	}
	if _, err = tx.ExecContext(ctx, `UPDATE couriers SET balance = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newBalance, courierID); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return newBalance, nil
}

func nullFloat64(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

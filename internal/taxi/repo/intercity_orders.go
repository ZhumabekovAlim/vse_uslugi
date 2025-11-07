package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// IntercityOrder represents a long-distance taxi request made as an advertisement.
type IntercityOrder struct {
	ID            int64
	PassengerID   sql.NullInt64
	DriverID      sql.NullInt64
	FromLocation  string
	ToLocation    string
	TripType      string
	Comment       sql.NullString
	Price         int
	ContactPhone  string
	DepartureDate time.Time
	DepartureTime sql.NullString
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      sql.NullTime
	CreatorRole   string

	DriverCarModel     sql.NullString
	DriverFullName     sql.NullString
	DriverRating       sql.NullFloat64
	DriverPhoto        sql.NullString
	DriverAvatar       sql.NullString
	DriverProfileStamp sql.NullTime

	PassengerFullName     sql.NullString
	PassengerAvatar       sql.NullString
	PassengerPhone        sql.NullString
	PassengerRating       sql.NullFloat64
	PassengerProfileStamp sql.NullTime
}

// IntercityOrdersRepo provides CRUD helpers for intercity taxi requests.
type IntercityOrdersRepo struct {
	db *sql.DB
}

// NewIntercityOrdersRepo constructs a repo instance.
func NewIntercityOrdersRepo(db *sql.DB) *IntercityOrdersRepo {
	return &IntercityOrdersRepo{db: db}
}

// Create inserts a new intercity order and returns its identifier.
func (r *IntercityOrdersRepo) Create(ctx context.Context, order IntercityOrder) (int64, error) {
	if order.CreatorRole == "" {
		order.CreatorRole = "passenger"
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO intercity_orders
(passenger_id, driver_id, creator_role, from_location, to_location, trip_type, comment, price, departure_date, departure_time, status)
VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		order.PassengerID,
		order.DriverID,
		order.CreatorRole,
		order.FromLocation,
		order.ToLocation,
		order.TripType,
		order.Comment,
		order.Price,
		order.DepartureDate,
		order.DepartureTime,
		order.Status,
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

// Get returns a single intercity order by id.
func (r *IntercityOrdersRepo) Get(ctx context.Context, id int64) (IntercityOrder, error) {
	row := r.db.QueryRowContext(ctx, `SELECT
io.id,
io.passenger_id,
io.driver_id,
io.from_location,
io.to_location,
io.trip_type,
io.comment,
io.price,
COALESCE(pu.phone, d.phone, '') AS contact_phone,
io.departure_date,
io.departure_time,
io.status,
io.created_at,
io.updated_at,
io.closed_at,
io.creator_role,
d.car_model,
CONCAT_WS(' ', du.surname, du.name, du.middlename) AS driver_full_name,
d.rating,
d.driver_photo,
du.avatar_path,
d.updated_at,
CONCAT_WS(' ', pu.surname, pu.name, pu.middlename) AS passenger_full_name,
pu.avatar_path AS passenger_avatar_path,
pu.phone AS passenger_phone,
pu.review_rating AS passenger_rating,
pu.updated_at AS passenger_profile_updated_at
FROM intercity_orders io
LEFT JOIN users pu ON io.passenger_id = pu.id
LEFT JOIN drivers d ON io.driver_id = d.id
LEFT JOIN users du ON d.user_id = du.id
WHERE io.id = ?`, id)
	var order IntercityOrder
	err := row.Scan(
		&order.ID,
		&order.PassengerID,
		&order.DriverID,
		&order.FromLocation,
		&order.ToLocation,
		&order.TripType,
		&order.Comment,
		&order.Price,
		&order.ContactPhone,
		&order.DepartureDate,
		&order.DepartureTime,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ClosedAt,
		&order.CreatorRole,
		&order.DriverCarModel,
		&order.DriverFullName,
		&order.DriverRating,
		&order.DriverPhoto,
		&order.DriverAvatar,
		&order.DriverProfileStamp,
		&order.PassengerFullName,
		&order.PassengerAvatar,
		&order.PassengerPhone,
		&order.PassengerRating,
		&order.PassengerProfileStamp,
	)
	if err != nil {
		return IntercityOrder{}, err
	}
	return order, nil
}

// IntercityOrdersFilter describes optional filters for listing orders.
type IntercityOrdersFilter struct {
	From        string
	To          string
	Date        *time.Time
	Time        *time.Time
	Status      string
	PassengerID int64
	DriverID    int64
	Limit       int
	Offset      int
}

// List returns orders matching the filter.
func (r *IntercityOrdersRepo) List(ctx context.Context, filter IntercityOrdersFilter) ([]IntercityOrder, error) {
	var (
		parts = []string{`
SELECT io.id, io.passenger_id, io.driver_id, io.from_location, io.to_location, io.trip_type, io.comment, io.price,
       COALESCE(pu.phone, d.phone, '') AS contact_phone, io.departure_date, io.departure_time, io.status,
       io.created_at, io.updated_at, io.closed_at, io.creator_role, d.car_model,
       CONCAT_WS(' ', du.surname, du.name, du.middlename) AS driver_full_name, d.rating, d.driver_photo, du.avatar_path,
       d.updated_at,
       CONCAT_WS(' ', pu.surname, pu.name, pu.middlename) AS passenger_full_name,
       pu.avatar_path AS passenger_avatar_path,
       pu.phone AS passenger_phone,
       pu.review_rating AS passenger_rating,
       pu.updated_at AS passenger_profile_updated_at
FROM intercity_orders io
LEFT JOIN users pu ON io.passenger_id = pu.id
LEFT JOIN drivers d ON io.driver_id = d.id
LEFT JOIN users du ON d.user_id = du.id`}
		where []string
		args  []interface{}
	)

	if filter.From != "" {
		where = append(where, "LOWER(io.from_location) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.From)+"%")
	}
	if filter.To != "" {
		where = append(where, "LOWER(io.to_location) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.To)+"%")
	}
	if filter.Date != nil {
		where = append(where, "io.departure_date = ?")
		args = append(args, filter.Date.Format("2006-01-02"))
	}
	if filter.Time != nil {
		where = append(where, "io.departure_time = ?")
		args = append(args, filter.Time.Format("15:04:05"))
	}
	if filter.Status != "" {
		where = append(where, "io.status = ?")
		args = append(args, filter.Status)
	}
	if filter.PassengerID > 0 {
		where = append(where, "io.passenger_id = ?")
		args = append(args, filter.PassengerID)
	}
	if filter.DriverID > 0 {
		where = append(where, "io.driver_id = ?")
		args = append(args, filter.DriverID)
	}
	if len(where) > 0 {
		parts = append(parts, "WHERE "+strings.Join(where, " AND "))
	}

	parts = append(parts, "ORDER BY io.departure_date ASC, io.departure_time ASC, io.created_at DESC")

	if filter.Limit > 0 {
		parts = append(parts, "LIMIT ?")
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		parts = append(parts, "OFFSET ?")
		args = append(args, filter.Offset)
	}

	query := strings.Join(parts, " ")
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []IntercityOrder
	for rows.Next() {
		var order IntercityOrder
		if err := rows.Scan(
			&order.ID,
			&order.PassengerID,
			&order.DriverID,
			&order.FromLocation,
			&order.ToLocation,
			&order.TripType,
			&order.Comment,
			&order.Price,
			&order.ContactPhone,
			&order.DepartureDate,
			&order.DepartureTime,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ClosedAt,
			&order.CreatorRole,
			&order.DriverCarModel,
			&order.DriverFullName,
			&order.DriverRating,
			&order.DriverPhoto,
			&order.DriverAvatar,
			&order.DriverProfileStamp,
			&order.PassengerFullName,
			&order.PassengerAvatar,
			&order.PassengerPhone,
			&order.PassengerRating,
			&order.PassengerProfileStamp,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// Close marks an order as closed by the passenger.
func (r *IntercityOrdersRepo) Close(ctx context.Context, id, passengerID int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE intercity_orders
SET status = 'closed', closed_at = NOW()
WHERE id = ? AND passenger_id = ? AND status = 'open'`, id, passengerID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

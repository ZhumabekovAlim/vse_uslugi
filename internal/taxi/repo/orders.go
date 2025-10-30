package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Order represents the orders table.
type Order struct {
	ID               int64
	PassengerID      int64
	DriverID         sql.NullInt64
	FromLon          float64
	FromLat          float64
	ToLon            float64
	ToLat            float64
	DistanceM        int
	EtaSeconds       int
	RecommendedPrice int
	ClientPrice      int
	PaymentMethod    string
	Status           string
	Notes            sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Addresses        []OrderAddress
}

// OrderAddress represents a waypoint for the order.
type OrderAddress struct {
	ID      int64
	OrderID int64
	Seq     int
	Lon     float64
	Lat     float64
	Address sql.NullString
}

// OrdersRepo provides access to orders data.
type OrdersRepo struct {
	db *sql.DB
}

// NewOrdersRepo constructs an OrdersRepo.
func NewOrdersRepo(db *sql.DB) *OrdersRepo {
	return &OrdersRepo{db: db}
}

// CreateWithDispatch creates an order and its dispatch record within a transaction.
func (r *OrdersRepo) CreateWithDispatch(ctx context.Context, order Order, dispatch DispatchRecord) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if len(order.Addresses) < 2 {
		return 0, fmt.Errorf("order must contain at least two addresses, got %d", len(order.Addresses))
	}

	res, err := tx.ExecContext(ctx, `INSERT INTO orders (passenger_id, from_lon, from_lat, to_lon, to_lat, distance_m, eta_s, recommended_price, client_price, payment_method, status, notes) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		order.PassengerID, order.FromLon, order.FromLat, order.ToLon, order.ToLat, order.DistanceM, order.EtaSeconds, order.RecommendedPrice, order.ClientPrice, order.PaymentMethod, "created", order.Notes)
	if err != nil {
		return 0, err
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err = insertOrderAddresses(ctx, tx, orderID, order.Addresses); err != nil {
		return 0, err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO order_dispatch (order_id, radius_m, next_tick_at, state) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE radius_m=VALUES(radius_m), next_tick_at=VALUES(next_tick_at), state=VALUES(state)`,
		orderID, dispatch.RadiusM, dispatch.NextTickAt, dispatch.State); err != nil {
		return 0, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE orders SET status = ? WHERE id = ?`, "searching", orderID); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

// Get fetches an order by id.
func (r *OrdersRepo) Get(ctx context.Context, id int64) (Order, error) {
	var o Order
	row := r.db.QueryRowContext(ctx, `SELECT id, passenger_id, driver_id, from_lon, from_lat, to_lon, to_lat, distance_m, eta_s, recommended_price, client_price, payment_method, status, notes, created_at, updated_at FROM orders WHERE id = ?`, id)
	err := row.Scan(&o.ID, &o.PassengerID, &o.DriverID, &o.FromLon, &o.FromLat, &o.ToLon, &o.ToLat, &o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod, &o.Status, &o.Notes, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return Order{}, err
	}

	o.Addresses, err = r.listAddresses(ctx, o.ID)
	if err != nil {
		return Order{}, err
	}
	return o, nil
}

func insertOrderAddresses(ctx context.Context, tx *sql.Tx, orderID int64, addresses []OrderAddress) error {
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO order_addresses (order_id, seq, lon, lat, address) VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, addr := range addresses {
		seq := addr.Seq
		if seq == 0 {
			seq = i
		}
		if _, err := stmt.ExecContext(ctx, orderID, seq, addr.Lon, addr.Lat, addr.Address); err != nil {
			return err
		}
	}
	return nil
}

func (r *OrdersRepo) listAddresses(ctx context.Context, orderID int64) ([]OrderAddress, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, seq, lon, lat, address FROM order_addresses WHERE order_id = ? ORDER BY seq ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []OrderAddress
	for rows.Next() {
		var addr OrderAddress
		if err := rows.Scan(&addr.ID, &addr.OrderID, &addr.Seq, &addr.Lon, &addr.Lat, &addr.Address); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return addresses, nil
}

// UpdatePrice updates the client price and logs history.
func (r *OrdersRepo) UpdatePrice(ctx context.Context, orderID int64, oldPrice, newPrice int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `UPDATE orders SET client_price = ? WHERE id = ?`, newPrice, orderID)
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

	if _, err = tx.ExecContext(ctx, `INSERT INTO order_price_history (order_id, old_price, new_price) VALUES (?,?,?)`, orderID, oldPrice, newPrice); err != nil {
		return err
	}

	return tx.Commit()
}

// AssignDriver assigns a driver to an order and updates status.
func (r *OrdersRepo) AssignDriver(ctx context.Context, orderID, driverID int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE orders SET driver_id = ?, status = ? WHERE id = ?`, driverID, "accepted", orderID)
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

// UpdateStatusCAS updates status when current status matches expected.
func (r *OrdersRepo) UpdateStatusCAS(ctx context.Context, orderID int64, fromStatus, toStatus string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`, toStatus, orderID, fromStatus)
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

// DispatchRecord represents order_dispatch row.
type DispatchRecord struct {
	ID         int64
	OrderID    int64
	RadiusM    int
	NextTickAt time.Time
	State      string
}

// DispatchRepo handles order_dispatch table.
type DispatchRepo struct {
	db *sql.DB
}

// NewDispatchRepo builds a dispatch repo.
func NewDispatchRepo(db *sql.DB) *DispatchRepo {
	return &DispatchRepo{db: db}
}

// ListDue returns dispatch records ready for processing.
func (r *DispatchRepo) ListDue(ctx context.Context, now time.Time) ([]DispatchRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, radius_m, next_tick_at, state FROM order_dispatch WHERE state = 'searching' AND next_tick_at <= ?`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DispatchRecord
	for rows.Next() {
		var rec DispatchRecord
		if err := rows.Scan(&rec.ID, &rec.OrderID, &rec.RadiusM, &rec.NextTickAt, &rec.State); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

// UpdateRadius updates radius and next tick.
func (r *DispatchRepo) UpdateRadius(ctx context.Context, orderID int64, radius int, next time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE order_dispatch SET radius_m = ?, next_tick_at = ? WHERE order_id = ?`, radius, next, orderID)
	return err
}

// MarkAssigned marks dispatch as assigned.
func (r *DispatchRepo) MarkAssigned(ctx context.Context, orderID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE order_dispatch SET state = 'assigned' WHERE order_id = ?`, orderID)
	return err
}

// Finish marks dispatch finished.
func (r *DispatchRepo) Finish(ctx context.Context, orderID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE order_dispatch SET state = 'finished' WHERE order_id = ?`, orderID)
	return err
}

// OffersRepo handles driver_order_offers.
type OffersRepo struct {
	db *sql.DB
}

// NewOffersRepo builds repo.
func NewOffersRepo(db *sql.DB) *OffersRepo { return &OffersRepo{db: db} }

// AlreadyOffered returns true if driver already has offer.
func (r *OffersRepo) AlreadyOffered(ctx context.Context, orderID, driverID int64) (bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT 1 FROM driver_order_offers WHERE order_id = ? AND driver_id = ?`, orderID, driverID)
	var x int
	err := row.Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateOffer inserts new offer if not exists.
func (r *OffersRepo) CreateOffer(ctx context.Context, orderID, driverID int64, ttl time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO driver_order_offers (order_id, driver_id, ttl_at) VALUES (?,?,?) ON DUPLICATE KEY UPDATE ttl_at = VALUES(ttl_at), state = 'pending'`, orderID, driverID, ttl)
	return err
}

// AcceptOffer sets offer as accepted and closes others.
func (r *OffersRepo) AcceptOffer(ctx context.Context, orderID, driverID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'accepted' WHERE order_id = ? AND driver_id = ? AND state = 'pending' AND ttl_at >= NOW()`, orderID, driverID)
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
	if _, err = tx.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'closed' WHERE order_id = ? AND driver_id <> ? AND state = 'pending'`, orderID, driverID); err != nil {
		return err
	}
	return tx.Commit()
}

// ExpireOffers marks offers as expired when TTL passed.
func (r *OffersRepo) ExpireOffers(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'expired' WHERE state = 'pending' AND ttl_at < ?`, now)
	return err
}

var ErrNoRows = errors.New("not found")

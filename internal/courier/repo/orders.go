package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
)

// Order statuses mirror courier lifecycle events.
const (
	StatusNew               = lifecycle.StatusNew
	StatusOffered           = lifecycle.StatusOffered
	StatusAssigned          = lifecycle.StatusAssigned
	StatusCourierArrived    = lifecycle.StatusCourierArrived
	StatusPickupStarted     = lifecycle.StatusPickupStarted
	StatusPickupDone        = lifecycle.StatusPickupDone
	StatusDeliveryStarted   = lifecycle.StatusDeliveryStarted
	StatusDelivered         = lifecycle.StatusDelivered
	StatusClosed            = lifecycle.StatusClosed
	StatusCanceledBySender  = lifecycle.StatusCanceledBySender
	StatusCanceledByCourier = lifecycle.StatusCanceledByCourier
	StatusCanceledNoShow    = lifecycle.StatusCanceledNoShow
)

// Order represents a courier order together with its route points.
type Order struct {
	ID               int64
	SenderID         int64
	CourierID        sql.NullInt64
	DistanceM        int
	EtaSeconds       int
	RecommendedPrice int
	ClientPrice      int
	PaymentMethod    string
	Status           string
	Comment          sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Points           []OrderPoint
}

// OrderPoint describes a delivery waypoint.
type OrderPoint struct {
	ID        int64
	OrderID   int64
	Seq       int
	Address   string
	Lat       float64
	Lon       float64
	Entrance  sql.NullString
	Apt       sql.NullString
	Floor     sql.NullString
	Intercom  sql.NullString
	Phone     sql.NullString
	Comment   sql.NullString
	CreatedAt time.Time
}

// StatusHistoryEntry captures a lifecycle change for auditing.
type StatusHistoryEntry struct {
	OrderID   int64
	Status    string
	Note      sql.NullString
	CreatedAt time.Time
}

// OrdersRepo provides persistence for courier orders and their points.
type OrdersRepo struct {
	db *sql.DB
}

// NewOrdersRepo constructs a new OrdersRepo.
func NewOrdersRepo(db *sql.DB) *OrdersRepo {
	return &OrdersRepo{db: db}
}

// Create inserts a new order with its route points.
func (r *OrdersRepo) Create(ctx context.Context, order Order) (int64, error) {
	if len(order.Points) < 2 {
		return 0, fmt.Errorf("order must contain at least two points")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, execErr := tx.ExecContext(ctx, `INSERT INTO courier_orders (sender_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment) VALUES (?,?,?,?,?,?,?,?)`,
		order.SenderID, order.DistanceM, order.EtaSeconds, order.RecommendedPrice, order.ClientPrice, order.PaymentMethod, StatusNew, nullOrString(order.Comment))
	if execErr != nil {
		err = execErr
		return 0, err
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err = insertOrderPoints(ctx, tx, orderID, order.Points); err != nil {
		return 0, err
	}

	if err = insertStatusHistory(ctx, tx, StatusHistoryEntry{OrderID: orderID, Status: StatusNew}); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

// Get returns an order with its points by identifier.
func (r *OrdersRepo) Get(ctx context.Context, id int64) (Order, error) {
	var o Order
	row := r.db.QueryRowContext(ctx, `SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE id = ?`, id)
	err := row.Scan(&o.ID, &o.SenderID, &o.CourierID, &o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod, &o.Status, &o.Comment, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	points, err := r.fetchPoints(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o.Points = points
	return o, nil
}

// ListBySender returns orders belonging to the sender.
func (r *OrdersRepo) ListBySender(ctx context.Context, senderID int64, limit, offset int) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE sender_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, senderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	if err = r.attachPoints(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// ListByCourier returns orders assigned to a courier.
func (r *OrdersRepo) ListByCourier(ctx context.Context, courierID int64, limit, offset int) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE courier_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, courierID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	if err = r.attachPoints(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// UpdateStatus moves an order to the next lifecycle state and records history.
func (r *OrdersRepo) UpdateStatus(ctx context.Context, orderID int64, nextStatus string, note sql.NullString) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var current string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM courier_orders WHERE id = ? FOR UPDATE`, orderID).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if current == nextStatus {
		return tx.Commit()
	}

	if !lifecycle.CanTransition(current, nextStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", current, nextStatus)
	}

	if _, err = tx.ExecContext(ctx, `UPDATE courier_orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, nextStatus, orderID); err != nil {
		return err
	}

	if err = insertStatusHistory(ctx, tx, StatusHistoryEntry{OrderID: orderID, Status: nextStatus, Note: note}); err != nil {
		return err
	}

	return tx.Commit()
}

// AssignCourier links a courier to the order and optionally moves it to a new status.
func (r *OrdersRepo) AssignCourier(ctx context.Context, orderID, courierID int64, nextStatus string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var current string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM courier_orders WHERE id = ? FOR UPDATE`, orderID).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	statusChanged := false
	if nextStatus != "" && current != nextStatus {
		if !lifecycle.CanTransition(current, nextStatus) {
			return fmt.Errorf("invalid status transition from %s to %s", current, nextStatus)
		}
		statusChanged = true
	}

	if statusChanged {
		if _, err = tx.ExecContext(ctx, `UPDATE courier_orders SET courier_id = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, courierID, nextStatus, orderID); err != nil {
			return err
		}
		if err = insertStatusHistory(ctx, tx, StatusHistoryEntry{OrderID: orderID, Status: nextStatus}); err != nil {
			return err
		}
	} else {
		if _, err = tx.ExecContext(ctx, `UPDATE courier_orders SET courier_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, courierID, orderID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrdersRepo) scanOrders(rows *sql.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.SenderID, &o.CourierID, &o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod, &o.Status, &o.Comment, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *OrdersRepo) attachPoints(ctx context.Context, orders []Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, 0, len(orders))
	orderIndex := make(map[int64]int, len(orders))
	for i, o := range orders {
		ids = append(ids, fmt.Sprintf("%d", o.ID))
		orderIndex[o.ID] = i
	}

	query := fmt.Sprintf(`SELECT id, order_id, seq, address, lat, lon, entrance, apt, floor, intercom, phone, comment, created_at FROM courier_order_points WHERE order_id IN (%s) ORDER BY order_id, seq`, strings.Join(ids, ","))
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var p OrderPoint
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Seq, &p.Address, &p.Lat, &p.Lon, &p.Entrance, &p.Apt, &p.Floor, &p.Intercom, &p.Phone, &p.Comment, &p.CreatedAt); err != nil {
			return err
		}
		if idx, ok := orderIndex[p.OrderID]; ok {
			orders[idx].Points = append(orders[idx].Points, p)
		}
	}
	return rows.Err()
}

func (r *OrdersRepo) fetchPoints(ctx context.Context, orderID int64) ([]OrderPoint, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, seq, address, lat, lon, entrance, apt, floor, intercom, phone, comment, created_at FROM courier_order_points WHERE order_id = ? ORDER BY seq`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []OrderPoint
	for rows.Next() {
		var p OrderPoint
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Seq, &p.Address, &p.Lat, &p.Lon, &p.Entrance, &p.Apt, &p.Floor, &p.Intercom, &p.Phone, &p.Comment, &p.CreatedAt); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

func insertOrderPoints(ctx context.Context, tx *sql.Tx, orderID int64, points []OrderPoint) error {
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO courier_order_points (order_id, seq, address, lat, lon, entrance, apt, floor, intercom, phone, comment) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, orderID, p.Seq, p.Address, p.Lat, p.Lon, nullOrString(p.Entrance), nullOrString(p.Apt), nullOrString(p.Floor), nullOrString(p.Intercom), nullOrString(p.Phone), nullOrString(p.Comment)); err != nil {
			return err
		}
	}
	return nil
}

func insertStatusHistory(ctx context.Context, tx *sql.Tx, entry StatusHistoryEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO courier_order_status_history (order_id, status, note, created_at) VALUES (?,?,?,?)`, entry.OrderID, entry.Status, nullOrString(entry.Note), entry.CreatedAt)
	return err
}

func nullOrString(val sql.NullString) interface{} {
	if val.Valid {
		return val.String
	}
	return nil
}

// UpdateStatusWithNote is a helper to pass raw string pointers from HTTP layer.
func (r *OrdersRepo) UpdateStatusWithNote(ctx context.Context, orderID int64, nextStatus string, note *string) error {
	var ns sql.NullString
	if note != nil {
		ns = sql.NullString{String: *note, Valid: true}
	}
	return r.UpdateStatus(ctx, orderID, nextStatus, ns)
}

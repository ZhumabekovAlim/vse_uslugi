package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"naimuBack/internal/taxi/fsm"
)

var (
	passengerActiveStatuses = []string{
		fsm.StatusSearching,
		fsm.StatusAccepted,
		fsm.StatusAssigned,
		fsm.StatusDriverAtPickup,
		fsm.StatusArrived,
		fsm.StatusWaitingFree,
		fsm.StatusWaitingPaid,
		fsm.StatusInProgress,
		fsm.StatusPickedUp,
		fsm.StatusAtLastPoint,
	}
	driverActiveStatuses = []string{
		fsm.StatusAccepted,
		fsm.StatusAssigned,
		fsm.StatusDriverAtPickup,
		fsm.StatusArrived,
		fsm.StatusWaitingFree,
		fsm.StatusWaitingPaid,
		fsm.StatusInProgress,
		fsm.StatusPickedUp,
		fsm.StatusAtLastPoint,
	}
	activePassengerQuery = fmt.Sprintf(
		`SELECT id FROM orders WHERE passenger_id = ? AND status IN (%s) ORDER BY created_at DESC LIMIT 1`,
		placeholders(len(passengerActiveStatuses)),
	)
	activeDriverQuery = fmt.Sprintf(
		`SELECT id FROM orders WHERE driver_id = ? AND status IN (%s) ORDER BY created_at DESC LIMIT 1`,
		placeholders(len(driverActiveStatuses)),
	)
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

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
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
	order, _, err := r.GetWithDriver(ctx, id)
	return order, err
}

// GetWithDriver fetches an order by id along with the assigned driver information.
func (r *OrdersRepo) GetWithDriver(ctx context.Context, id int64) (Order, *Driver, error) {
	var (
		o Order

		driverID           sql.NullInt64
		driverUserID       sql.NullInt64
		driverStatus       sql.NullString
		driverCarModel     sql.NullString
		driverCarColor     sql.NullString
		driverCarNumber    sql.NullString
		driverTechPassport sql.NullString
		driverPhotoFront   sql.NullString
		driverPhotoBack    sql.NullString
		driverPhotoLeft    sql.NullString
		driverPhotoRight   sql.NullString
		driverDriverPhoto  sql.NullString
		driverPhone        sql.NullString
		driverIIN          sql.NullString
		driverIDCardFront  sql.NullString
		driverIDCardBack   sql.NullString
		driverRating       sql.NullFloat64
		driverUpdatedAt    sql.NullTime
		driverName         sql.NullString
		driverSurname      sql.NullString
		driverMiddlename   sql.NullString
	)

	row := r.db.QueryRowContext(ctx, `SELECT
        o.id, o.passenger_id, o.driver_id, o.from_lon, o.from_lat, o.to_lon, o.to_lat,
        o.distance_m, o.eta_s, o.recommended_price, o.client_price, o.payment_method,
        o.status, o.notes, o.created_at, o.updated_at,
        d.id, d.user_id, d.status, d.car_model, d.car_color, d.car_number,
        d.tech_passport, d.car_photo_front, d.car_photo_back, d.car_photo_left, d.car_photo_right,
        d.driver_photo, d.phone, d.iin, d.id_card_front, d.id_card_back, d.rating, d.updated_at,
        u.name, u.surname, u.middlename
    FROM orders o
    LEFT JOIN drivers d ON d.id = o.driver_id
    LEFT JOIN users u ON u.id = d.user_id
    WHERE o.id = ?`, id)
	err := row.Scan(
		&o.ID, &o.PassengerID, &o.DriverID, &o.FromLon, &o.FromLat, &o.ToLon, &o.ToLat,
		&o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod,
		&o.Status, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		&driverID, &driverUserID, &driverStatus, &driverCarModel, &driverCarColor, &driverCarNumber,
		&driverTechPassport, &driverPhotoFront, &driverPhotoBack, &driverPhotoLeft, &driverPhotoRight,
		&driverDriverPhoto, &driverPhone, &driverIIN, &driverIDCardFront, &driverIDCardBack, &driverRating, &driverUpdatedAt,
		&driverName, &driverSurname, &driverMiddlename,
	)
	if err != nil {
		return Order{}, nil, err
	}

	o.Addresses, err = r.listAddresses(ctx, o.ID)
	if err != nil {
		return Order{}, nil, err
	}

	if !driverID.Valid {
		return o, nil, nil
	}

	driver := Driver{
		ID:       driverID.Int64,
		CarModel: driverCarModel,
		CarColor: driverCarColor,
	}
	if driverUserID.Valid {
		driver.UserID = driverUserID.Int64
	}
	if driverName.Valid {
		driver.Name = driverName.String
	}
	if driverSurname.Valid {
		driver.Surname = driverSurname.String
	}
	if driverMiddlename.Valid {
		driver.Middlename = driverMiddlename
	}
	if driverStatus.Valid {
		driver.Status = driverStatus.String
	}
	if driverCarNumber.Valid {
		driver.CarNumber = driverCarNumber.String
	}
	if driverTechPassport.Valid {
		driver.TechPassport = driverTechPassport.String
	}
	if driverPhotoFront.Valid {
		driver.CarPhotoFront = driverPhotoFront.String
	}
	if driverPhotoBack.Valid {
		driver.CarPhotoBack = driverPhotoBack.String
	}
	if driverPhotoLeft.Valid {
		driver.CarPhotoLeft = driverPhotoLeft.String
	}
	if driverPhotoRight.Valid {
		driver.CarPhotoRight = driverPhotoRight.String
	}
	if driverDriverPhoto.Valid {
		driver.DriverPhoto = driverDriverPhoto.String
	}
	if driverPhone.Valid {
		driver.Phone = driverPhone.String
	}
	if driverIIN.Valid {
		driver.IIN = driverIIN.String
	}
	if driverIDCardFront.Valid {
		driver.IDCardFront = driverIDCardFront.String
	}
	if driverIDCardBack.Valid {
		driver.IDCardBack = driverIDCardBack.String
	}
	if driverRating.Valid {
		driver.Rating = driverRating.Float64
	}
	if driverUpdatedAt.Valid {
		driver.UpdatedAt = driverUpdatedAt.Time
	}

	return o, &driver, nil
}

// ListByPassenger returns orders belonging to passenger sorted by creation date.
func (r *OrdersRepo) ListByPassenger(ctx context.Context, passengerID int64, limit, offset int) ([]Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, passenger_id, driver_id, from_lon, from_lat, to_lon, to_lat, distance_m, eta_s, recommended_price, client_price, payment_method, status, notes, created_at, updated_at FROM orders WHERE passenger_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, passengerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.PassengerID, &o.DriverID, &o.FromLon, &o.FromLat, &o.ToLon, &o.ToLat, &o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod, &o.Status, &o.Notes, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		addresses, err := r.listAddresses(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Addresses = addresses
	}
	return orders, nil
}

// ListByDriver returns orders assigned to a driver sorted by creation date.
func (r *OrdersRepo) ListByDriver(ctx context.Context, driverID int64, limit, offset int) ([]Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.QueryContext(ctx, `SELECT id, passenger_id, driver_id, from_lon, from_lat, to_lon, to_lat, distance_m,
 eta_s, recommended_price, client_price, payment_method, status, notes, created_at, updated_at FROM orders WHERE driver_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, driverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.PassengerID, &o.DriverID, &o.FromLon, &o.FromLat, &o.ToLon, &o.ToLat, &o.DistanceM, &o.EtaSeconds, &o.RecommendedPrice, &o.ClientPrice, &o.PaymentMethod, &o.Status, &o.Notes, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		addresses, err := r.listAddresses(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Addresses = addresses
	}
	return orders, nil
}

// GetActiveOrderIDByPassenger returns the most recent active order ID for the passenger.
func (r *OrdersRepo) GetActiveOrderIDByPassenger(ctx context.Context, passengerID int64) (int64, error) {
	args := make([]interface{}, 0, len(passengerActiveStatuses)+1)
	args = append(args, passengerID)
	for _, status := range passengerActiveStatuses {
		args = append(args, status)
	}

	var orderID int64
	if err := r.db.QueryRowContext(ctx, activePassengerQuery, args...).Scan(&orderID); err != nil {
		return 0, err
	}
	return orderID, nil
}

// GetActiveOrderIDByDriver returns the most recent active order ID for the driver.
func (r *OrdersRepo) GetActiveOrderIDByDriver(ctx context.Context, driverID int64) (int64, error) {
	args := make([]interface{}, 0, len(driverActiveStatuses)+1)
	args = append(args, driverID)
	for _, status := range driverActiveStatuses {
		args = append(args, status)
	}

	var orderID int64
	if err := r.db.QueryRowContext(ctx, activeDriverQuery, args...).Scan(&orderID); err != nil {
		return 0, err
	}
	return orderID, nil
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
	CreatedAt  time.Time
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, radius_m, next_tick_at, state, created_at FROM order_dispatch WHERE state = 'searching' AND next_tick_at <= ?`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DispatchRecord
	for rows.Next() {
		var rec DispatchRecord
		if err := rows.Scan(&rec.ID, &rec.OrderID, &rec.RadiusM, &rec.NextTickAt, &rec.State, &rec.CreatedAt); err != nil {
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

// SetDriverPrice stores driver's proposed price for an offer.
func (r *OffersRepo) SetDriverPrice(ctx context.Context, orderID, driverID int64, price int) error {
	res, err := r.db.ExecContext(ctx, `UPDATE driver_order_offers SET driver_price = ?, state = 'pending' WHERE order_id = ? AND driver_id = ? AND state IN ('pending','declined')`, price, orderID, driverID)
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

// AcceptOffer sets offer as accepted and closes others.
func (r *OffersRepo) AcceptOffer(ctx context.Context, orderID, driverID int64) ([]int64, *int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var price sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT driver_price FROM driver_order_offers WHERE order_id = ? AND driver_id = ? AND state = 'pending' FOR UPDATE`, orderID, driverID).Scan(&price); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, sql.ErrNoRows
		}
		return nil, nil, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'accepted' WHERE order_id = ? AND driver_id = ? AND state = 'pending'`, orderID, driverID); err != nil {
		return nil, nil, err
	}

	rowsClosed, err := tx.QueryContext(ctx, `SELECT driver_id FROM driver_order_offers WHERE order_id = ? AND driver_id <> ? AND state = 'pending'`, orderID, driverID)
	if err != nil {
		return nil, nil, err
	}
	defer rowsClosed.Close()

	var closedDrivers []int64
	for rowsClosed.Next() {
		var id int64
		if err = rowsClosed.Scan(&id); err != nil {
			return nil, nil, err
		}
		closedDrivers = append(closedDrivers, id)
	}
	if err = rowsClosed.Err(); err != nil {
		return nil, nil, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'closed' WHERE order_id = ? AND driver_id <> ? AND state = 'pending'`, orderID, driverID); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}

	if price.Valid {
		value := int(price.Int64)
		return closedDrivers, &value, nil
	}
	return closedDrivers, nil, nil
}

// DeclineOffer marks offer as declined.
func (r *OffersRepo) DeclineOffer(ctx context.Context, orderID, driverID int64) (*int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var price sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT driver_price FROM driver_order_offers WHERE order_id = ? AND driver_id = ? AND state = 'pending' FOR UPDATE`, orderID, driverID).Scan(&price); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'declined' WHERE order_id = ? AND driver_id = ? AND state = 'pending'`, orderID, driverID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	if price.Valid {
		value := int(price.Int64)
		return &value, nil
	}
	return nil, nil
}

// ExpireOffers marks offers as expired when TTL passed.
func (r *OffersRepo) ExpireOffers(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE driver_order_offers SET state = 'expired' WHERE state = 'pending' AND ttl_at < ?`, now)
	return err
}

var ErrNoRows = errors.New("not found")

// GetActiveOfferDriverIDs возвращает id водителей, у кого по заказу висит актуальный оффер.
// GetActiveOfferDriverIDs возвращает ID водителей с актуальными офферами по заказу.
func (r *OffersRepo) GetActiveOfferDriverIDs(ctx context.Context, orderID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT driver_id
		FROM driver_order_offers
		WHERE order_id = ?
		  AND state = 'pending'
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

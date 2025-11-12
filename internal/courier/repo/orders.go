package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
)

var (
	activeStatuses = []string{
		StatusNew,
		StatusAccepted,
		StatusWaitingFree,
		StatusInProgress,
	}
	completedStatuses = []string{
		StatusCompleted,
		StatusClosed,
	}
	canceledStatuses = []string{
		StatusCanceledBySender,
		StatusCanceledByCourier,
		StatusCanceledNoShow,
	}
	activeStatusSet    = statusSet(activeStatuses)
	completedStatusSet = statusSet(completedStatuses)
	canceledStatusSet  = statusSet(canceledStatuses)
)

var (
	ErrReviewForbidden        = errors.New("courier review forbidden")
	ErrReviewOrderNotFinished = errors.New("courier order not completed")
	ErrReviewCourierMissing   = errors.New("courier not assigned")
)

// Order statuses mirror courier lifecycle events.
const (
	StatusNew               = lifecycle.StatusNew
	StatusAccepted          = lifecycle.StatusAccepted
	StatusWaitingFree       = lifecycle.StatusWaitingFree
	StatusInProgress        = lifecycle.StatusInProgress
	StatusCompleted         = lifecycle.StatusCompleted
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
	Sender           User
	Courier          *Courier
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

// DispatchRecord represents a courier order dispatch entry.
type DispatchRecord struct {
	ID         int64
	OrderID    int64
	RadiusM    int
	NextTickAt time.Time
	State      string
	CreatedAt  time.Time
}

// CourierReview holds feedback exchanged between sender and courier.
type CourierReview struct {
	ID            int64
	SenderRating  sql.NullFloat64
	CourierRating sql.NullFloat64
	Comment       sql.NullString
	CreatedAt     time.Time
	Order         Order
}

// OrdersStats aggregates counts for admin dashboards.
type OrdersStats struct {
	Total     int `json:"total_orders"`
	Active    int `json:"active_orders"`
	Completed int `json:"completed_orders"`
	Canceled  int `json:"canceled_orders"`
}

// CourierOrderStats aggregates orders metrics per courier profile.
type CourierOrderStats struct {
	Total     int `json:"total_orders"`
	Active    int `json:"active_orders"`
	Completed int `json:"completed_orders"`
	Canceled  int `json:"canceled_orders"`
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
	return r.create(ctx, order, nil)
}

// CreateWithDispatch inserts a new order together with a dispatch record within a single transaction.
func (r *OrdersRepo) CreateWithDispatch(ctx context.Context, order Order, dispatch DispatchRecord) (int64, error) {
	return r.create(ctx, order, &dispatch)
}

func (r *OrdersRepo) create(ctx context.Context, order Order, dispatch *DispatchRecord) (orderID int64, err error) {
	if len(order.Points) < 2 {
		return 0, fmt.Errorf("order must contain at least two points")
	}
	tx, beginErr := r.db.BeginTx(ctx, nil)
	if beginErr != nil {
		return 0, beginErr
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
	orderID, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err = insertOrderPoints(ctx, tx, orderID, order.Points); err != nil {
		return 0, err
	}

	if err = insertStatusHistory(ctx, tx, StatusHistoryEntry{OrderID: orderID, Status: StatusNew}); err != nil {
		return 0, err
	}

	if dispatch != nil {
		if err = insertDispatchRecord(ctx, tx, orderID, *dispatch); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

// Get returns an order with its points by identifier.
func (r *OrdersRepo) Get(ctx context.Context, id int64) (Order, error) {
	var (
		o                Order
		courierUserID    sql.NullInt64
		courierFirstName sql.NullString
		courierLastName  sql.NullString
		courierMiddle    sql.NullString
		courierPhoto     sql.NullString
		courierIIN       sql.NullString
		courierBirthDate sql.NullTime
		courierCardFront sql.NullString
		courierCardBack  sql.NullString
		courierPhone     sql.NullString
		courierRating    sql.NullFloat64
		courierBalance   sql.NullInt64
		courierStatus    sql.NullString
		courierCreatedAt sql.NullTime
		courierUpdatedAt sql.NullTime
	)
	row := r.db.QueryRowContext(ctx, `
        SELECT
                o.id,
                o.sender_id,
                o.courier_id,
                o.distance_m,
                o.eta_seconds,
                o.recommended_price,
                o.client_price,
                o.payment_method,
                o.status,
                o.comment,
                o.created_at,
                o.updated_at,
                s.id,
                s.name,
                s.surname,
                s.middlename,
                s.phone,
                s.email,
                s.city_id,
                s.years_of_exp,
                s.doc_of_proof,
                s.review_rating,
                s.role,
                s.latitude,
                s.longitude,
                s.avatar_path,
                s.skills,
                s.is_online,
                s.created_at,
                s.updated_at,
                c.user_id,
                c.first_name,
                c.last_name,
                c.middle_name,
                c.courier_photo,
                c.iin,
                c.date_of_birth,
                c.id_card_front,
                c.id_card_back,
                c.phone,
                c.rating,
                c.balance,
                c.status,
                c.created_at,
                c.updated_at
        FROM courier_orders o
        JOIN users s ON s.id = o.sender_id
        LEFT JOIN couriers c ON c.id = o.courier_id
        WHERE o.id = ?`, id)
	err := row.Scan(
		&o.ID,
		&o.SenderID,
		&o.CourierID,
		&o.DistanceM,
		&o.EtaSeconds,
		&o.RecommendedPrice,
		&o.ClientPrice,
		&o.PaymentMethod,
		&o.Status,
		&o.Comment,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Sender.ID,
		&o.Sender.Name,
		&o.Sender.Surname,
		&o.Sender.Middlename,
		&o.Sender.Phone,
		&o.Sender.Email,
		&o.Sender.CityID,
		&o.Sender.YearsOfExp,
		&o.Sender.DocOfProof,
		&o.Sender.ReviewRating,
		&o.Sender.Role,
		&o.Sender.Latitude,
		&o.Sender.Longitude,
		&o.Sender.AvatarPath,
		&o.Sender.Skills,
		&o.Sender.IsOnline,
		&o.Sender.CreatedAt,
		&o.Sender.UpdatedAt,
		&courierUserID,
		&courierFirstName,
		&courierLastName,
		&courierMiddle,
		&courierPhoto,
		&courierIIN,
		&courierBirthDate,
		&courierCardFront,
		&courierCardBack,
		&courierPhone,
		&courierRating,
		&courierBalance,
		&courierStatus,
		&courierCreatedAt,
		&courierUpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	if o.CourierID.Valid && courierUserID.Valid {
		courier := Courier{
			ID:          o.CourierID.Int64,
			UserID:      courierUserID.Int64,
			FirstName:   courierFirstName.String,
			LastName:    courierLastName.String,
			MiddleName:  courierMiddle,
			Photo:       courierPhoto.String,
			IIN:         courierIIN.String,
			IDCardFront: courierCardFront.String,
			IDCardBack:  courierCardBack.String,
			Phone:       courierPhone.String,
			Rating:      courierRating,
			Balance:     int(courierBalance.Int64),
			Status:      courierStatus.String,
		}
		if courierBirthDate.Valid {
			courier.BirthDate = courierBirthDate.Time
		}
		if courierCreatedAt.Valid {
			courier.CreatedAt = courierCreatedAt.Time
		}
		if courierUpdatedAt.Valid {
			courier.UpdatedAt = courierUpdatedAt.Time
		}
		o.Courier = &courier
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE sender_id = ? AND status = 'completed' ORDER BY created_at DESC LIMIT ? OFFSET ?`, senderID, limit, offset)
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

// ListCourierReviews returns reviews for a courier with latest feedback first.
func (r *OrdersRepo) ListCourierReviews(ctx context.Context, courierID int64) ([]CourierReview, error) {
	if courierID <= 0 {
		return nil, errors.New("invalid courier id")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT
        cr.id,
        cr.rating,
        cr.comment,
        cr.courier_rating,
        cr.created_at,
        o.id,
        o.sender_id,
        o.courier_id,
        o.distance_m,
        o.eta_seconds,
        o.recommended_price,
        o.client_price,
        o.payment_method,
        o.status,
        o.comment,
        o.created_at,
        o.updated_at
    FROM courier_reviews cr
    JOIN courier_orders o ON o.id = cr.order_id
    WHERE cr.courier_id = ?
    ORDER BY cr.created_at DESC`, courierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []CourierReview
	for rows.Next() {
		var review CourierReview
		if err := rows.Scan(
			&review.ID,
			&review.SenderRating,
			&review.Comment,
			&review.CourierRating,
			&review.CreatedAt,
			&review.Order.ID,
			&review.Order.SenderID,
			&review.Order.CourierID,
			&review.Order.DistanceM,
			&review.Order.EtaSeconds,
			&review.Order.RecommendedPrice,
			&review.Order.ClientPrice,
			&review.Order.PaymentMethod,
			&review.Order.Status,
			&review.Order.Comment,
			&review.Order.CreatedAt,
			&review.Order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(reviews) == 0 {
		return reviews, nil
	}

	orders := make([]Order, len(reviews))
	for i := range reviews {
		orders[i] = reviews[i].Order
	}
	if err := r.attachPoints(ctx, orders); err != nil {
		return nil, err
	}
	for i := range reviews {
		reviews[i].Order.Points = orders[i].Points
	}

	return reviews, nil
}

// ListByCourier returns orders assigned to a courier.
func (r *OrdersRepo) ListByCourier(ctx context.Context, courierID int64, limit, offset int) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE courier_id = ? AND status = 'completed'  ORDER BY created_at DESC LIMIT ? OFFSET ?`, courierID, limit, offset)
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

// ListCompletedByCourierBetween returns completed courier orders in the [from, to) interval based on updated_at.
func (r *OrdersRepo) ListCompletedByCourierBetween(ctx context.Context, courierID int64, from, to time.Time) ([]Order, error) {
	if courierID <= 0 {
		return nil, errors.New("invalid courier id")
	}
	if to.Before(from) {
		return nil, errors.New("invalid date range")
	}

	args := make([]interface{}, 0, len(completedStatuses)+3)
	args = append(args, courierID)
	for _, status := range completedStatuses {
		args = append(args, status)
	}
	args = append(args, from, to)

	query := fmt.Sprintf(`SELECT id, sender_id, courier_id, distance_m, eta_seconds, recommended_price, client_price, payment_method, status, comment, created_at, updated_at FROM courier_orders WHERE courier_id = ? AND status IN (%s) AND updated_at >= ? AND updated_at < ? ORDER BY updated_at ASC`, placeholders(len(completedStatuses)))

	rows, err := r.db.QueryContext(ctx, query, args...)
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

// SetSenderReview records sender feedback and refreshes courier rating.
func (r *OrdersRepo) SetSenderReview(ctx context.Context, orderID, senderID int64, rating *float64, comment *string) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		dbSender  int64
		dbCourier sql.NullInt64
		status    string
	)
	if err = tx.QueryRowContext(ctx, `SELECT sender_id, courier_id, status FROM courier_orders WHERE id = ? FOR UPDATE`, orderID).
		Scan(&dbSender, &dbCourier, &status); err != nil {
		return err
	}
	if dbSender != senderID {
		return ErrReviewForbidden
	}
	if _, ok := completedStatusSet[status]; !ok {
		return ErrReviewOrderNotFinished
	}
	if !dbCourier.Valid {
		return ErrReviewCourierMissing
	}

	var ratingValue interface{}
	if rating != nil {
		ratingValue = clampRating(*rating)
	}
	var commentValue interface{}
	if comment != nil {
		commentValue = *comment
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO courier_reviews (order_id, courier_id, rating, comment) VALUES (?,?,?,?)
        ON DUPLICATE KEY UPDATE rating = VALUES(rating), comment = VALUES(comment), created_at = CURRENT_TIMESTAMP`,
		orderID, dbCourier.Int64, ratingValue, commentValue); err != nil {
		return err
	}

	if err = updateCourierAverageRatingTx(ctx, tx, dbCourier.Int64); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// SetCourierReview records courier feedback about the sender and refreshes sender rating.
func (r *OrdersRepo) SetCourierReview(ctx context.Context, orderID, courierID int64, rating *float64) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		dbSender  int64
		dbCourier sql.NullInt64
		status    string
	)
	if err = tx.QueryRowContext(ctx, `SELECT sender_id, courier_id, status FROM courier_orders WHERE id = ? FOR UPDATE`, orderID).
		Scan(&dbSender, &dbCourier, &status); err != nil {
		return err
	}
	if !dbCourier.Valid || dbCourier.Int64 != courierID {
		return ErrReviewForbidden
	}
	if _, ok := completedStatusSet[status]; !ok {
		return ErrReviewOrderNotFinished
	}

	var ratingValue interface{}
	if rating != nil {
		ratingValue = clampRating(*rating)
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO courier_reviews (order_id, courier_id, courier_rating) VALUES (?,?,?)
        ON DUPLICATE KEY UPDATE courier_rating = VALUES(courier_rating), created_at = CURRENT_TIMESTAMP`,
		orderID, courierID, ratingValue); err != nil {
		return err
	}

	if err = updateSenderAverageRatingTx(ctx, tx, dbSender); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func updateCourierAverageRatingTx(ctx context.Context, tx *sql.Tx, courierID int64) error {
	var avg sql.NullFloat64
	if err := tx.QueryRowContext(ctx, `SELECT AVG(rating) FROM courier_reviews WHERE courier_id = ? AND rating IS NOT NULL`, courierID).Scan(&avg); err != nil {
		return err
	}
	var value interface{}
	if avg.Valid {
		value = clampRating(avg.Float64)
	}
	_, err := tx.ExecContext(ctx, `UPDATE couriers SET rating = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, value, courierID)
	return err
}

func updateSenderAverageRatingTx(ctx context.Context, tx *sql.Tx, senderID int64) error {
	var avg sql.NullFloat64
	if err := tx.QueryRowContext(ctx, `SELECT AVG(cr.courier_rating)
        FROM courier_reviews cr
        JOIN courier_orders o ON o.id = cr.order_id
        WHERE o.sender_id = ? AND cr.courier_rating IS NOT NULL`, senderID).Scan(&avg); err != nil {
		return err
	}
	var value interface{}
	if avg.Valid {
		value = clampRating(avg.Float64)
	}
	_, err := tx.ExecContext(ctx, `UPDATE users SET review_rating = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, value, senderID)
	return err
}

func clampRating(v float64) float64 {
	if v < 0 {
		v = 0
	}
	if v > 5 {
		v = 5
	}
	return math.Round(v*100) / 100
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

func insertDispatchRecord(ctx context.Context, tx *sql.Tx, orderID int64, dispatch DispatchRecord) error {
	if dispatch.State == "" {
		dispatch.State = StatusNew
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO courier_order_dispatch (order_id, radius_m, next_tick_at, state) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE radius_m=VALUES(radius_m), next_tick_at=VALUES(next_tick_at), state=VALUES(state)`,
		orderID, dispatch.RadiusM, dispatch.NextTickAt, dispatch.State)
	return err
}

// DispatchRepo provides access to courier_order_dispatch table.
type DispatchRepo struct {
	db *sql.DB
}

// NewDispatchRepo constructs DispatchRepo.
func NewDispatchRepo(db *sql.DB) *DispatchRepo {
	return &DispatchRepo{db: db}
}

// ListDue returns dispatch records ready for processing.
func (r *DispatchRepo) ListDue(ctx context.Context, now time.Time) ([]DispatchRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, radius_m, next_tick_at, state, created_at FROM courier_order_dispatch WHERE state = 'searching' AND next_tick_at <= ?`, now)
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

// UpdateRadius updates dispatch radius and schedules the next tick.
func (r *DispatchRepo) UpdateRadius(ctx context.Context, orderID int64, radius int, next time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE courier_order_dispatch SET radius_m = ?, next_tick_at = ? WHERE order_id = ?`, radius, next, orderID)
	return err
}

// Finish marks dispatch as completed.
func (r *DispatchRepo) Finish(ctx context.Context, orderID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE courier_order_dispatch SET state = 'finished' WHERE order_id = ?`, orderID)
	return err
}

// TriggerImmediate sets the next tick to the provided moment without changing radius.
func (r *DispatchRepo) TriggerImmediate(ctx context.Context, orderID int64, next time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE courier_order_dispatch SET next_tick_at = ? WHERE order_id = ?`, next, orderID)
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

// ActiveBySender returns the latest active order for sender.
func (r *OrdersRepo) ActiveBySender(ctx context.Context, senderID int64) (Order, error) {
	if senderID == 0 {
		return Order{}, fmt.Errorf("sender id required")
	}
	query := fmt.Sprintf(`SELECT id FROM courier_orders WHERE sender_id = ? AND status IN (%s) ORDER BY created_at DESC LIMIT 1`, placeholders(len(activeStatuses)))
	args := make([]interface{}, 0, len(activeStatuses)+1)
	args = append(args, senderID)
	for _, st := range activeStatuses {
		args = append(args, st)
	}
	var orderID int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&orderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, err
	}
	return r.Get(ctx, orderID)
}

// ActiveByCourier returns the latest active order for courier.
func (r *OrdersRepo) ActiveByCourier(ctx context.Context, courierID int64) (Order, error) {
	if courierID == 0 {
		return Order{}, fmt.Errorf("courier id required")
	}
	query := fmt.Sprintf(`SELECT id FROM courier_orders WHERE courier_id = ? AND status IN (%s) ORDER BY created_at DESC LIMIT 1`, placeholders(len(activeStatuses)))
	args := make([]interface{}, 0, len(activeStatuses)+1)
	args = append(args, courierID)
	for _, st := range activeStatuses {
		args = append(args, st)
	}
	var orderID int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&orderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, err
	}
	return r.Get(ctx, orderID)
}

// ListAll returns orders for admin usage.
func (r *OrdersRepo) ListAll(ctx context.Context, limit, offset int) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
        o.id,
        o.sender_id,
        o.courier_id,
        o.distance_m,
        o.eta_seconds,
        o.recommended_price,
        o.client_price,
        o.payment_method,
        o.status,
        o.comment,
        o.created_at,
        o.updated_at,
        s.id,
        s.name,
        s.surname,
        s.middlename,
        s.phone,
        s.email,
        s.city_id,
        s.years_of_exp,
        s.doc_of_proof,
        s.review_rating,
        s.role,
        s.latitude,
        s.longitude,
        s.avatar_path,
        s.skills,
        s.is_online,
        s.created_at,
        s.updated_at,
        c.user_id,
        c.first_name,
        c.last_name,
        c.middle_name,
        c.courier_photo,
        c.iin,
        c.date_of_birth,
        c.id_card_front,
        c.id_card_back,
        c.phone,
        c.rating,
        c.balance,
        c.status,
        c.created_at,
        c.updated_at
FROM courier_orders o
JOIN users s ON s.id = o.sender_id
LEFT JOIN couriers c ON c.id = o.courier_id
ORDER BY o.created_at DESC
LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var (
			o                Order
			courierUserID    sql.NullInt64
			courierFirstName sql.NullString
			courierLastName  sql.NullString
			courierMiddle    sql.NullString
			courierPhoto     sql.NullString
			courierIIN       sql.NullString
			courierBirthDate sql.NullTime
			courierCardFront sql.NullString
			courierCardBack  sql.NullString
			courierPhone     sql.NullString
			courierRating    sql.NullFloat64
			courierBalance   sql.NullInt64
			courierStatus    sql.NullString
			courierCreatedAt sql.NullTime
			courierUpdatedAt sql.NullTime
		)
		if err := rows.Scan(
			&o.ID,
			&o.SenderID,
			&o.CourierID,
			&o.DistanceM,
			&o.EtaSeconds,
			&o.RecommendedPrice,
			&o.ClientPrice,
			&o.PaymentMethod,
			&o.Status,
			&o.Comment,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Sender.ID,
			&o.Sender.Name,
			&o.Sender.Surname,
			&o.Sender.Middlename,
			&o.Sender.Phone,
			&o.Sender.Email,
			&o.Sender.CityID,
			&o.Sender.YearsOfExp,
			&o.Sender.DocOfProof,
			&o.Sender.ReviewRating,
			&o.Sender.Role,
			&o.Sender.Latitude,
			&o.Sender.Longitude,
			&o.Sender.AvatarPath,
			&o.Sender.Skills,
			&o.Sender.IsOnline,
			&o.Sender.CreatedAt,
			&o.Sender.UpdatedAt,
			&courierUserID,
			&courierFirstName,
			&courierLastName,
			&courierMiddle,
			&courierPhoto,
			&courierIIN,
			&courierBirthDate,
			&courierCardFront,
			&courierCardBack,
			&courierPhone,
			&courierRating,
			&courierBalance,
			&courierStatus,
			&courierCreatedAt,
			&courierUpdatedAt,
		); err != nil {
			return nil, err
		}
		if o.CourierID.Valid && courierUserID.Valid {
			courier := Courier{
				ID:          o.CourierID.Int64,
				UserID:      courierUserID.Int64,
				FirstName:   courierFirstName.String,
				LastName:    courierLastName.String,
				MiddleName:  courierMiddle,
				Photo:       courierPhoto.String,
				IIN:         courierIIN.String,
				IDCardFront: courierCardFront.String,
				IDCardBack:  courierCardBack.String,
				Phone:       courierPhone.String,
				Rating:      courierRating,
				Balance:     int(courierBalance.Int64),
				Status:      courierStatus.String,
			}
			if courierBirthDate.Valid {
				courier.BirthDate = courierBirthDate.Time
			}
			if courierCreatedAt.Valid {
				courier.CreatedAt = courierCreatedAt.Time
			}
			if courierUpdatedAt.Valid {
				courier.UpdatedAt = courierUpdatedAt.Time
			}
			o.Courier = &courier
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err = r.attachPoints(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// Stats returns aggregated counts across courier orders.
func (r *OrdersRepo) Stats(ctx context.Context) (OrdersStats, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM courier_orders GROUP BY status`)
	if err != nil {
		return OrdersStats{}, err
	}
	defer rows.Close()

	var stats OrdersStats
	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return OrdersStats{}, err
		}
		stats.Total += count
		if _, ok := activeStatusSet[status]; ok {
			stats.Active += count
		}
		if _, ok := completedStatusSet[status]; ok {
			stats.Completed += count
		}
		if _, ok := canceledStatusSet[status]; ok {
			stats.Canceled += count
		}
	}
	if err := rows.Err(); err != nil {
		return OrdersStats{}, err
	}
	return stats, nil
}

// UpdatePrice overwrites the client price for the order.
func (r *OrdersRepo) UpdatePrice(ctx context.Context, orderID int64, newPrice int) error {
	res, err := r.db.ExecContext(ctx, `UPDATE courier_orders SET client_price = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newPrice, orderID)
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

// UpdateStatusCAS updates order status if current matches expected state.
func (r *OrdersRepo) UpdateStatusCAS(ctx context.Context, orderID int64, current, next string) error {
	if current == next {
		return nil
	}
	if !lifecycle.CanTransition(current, next) {
		return fmt.Errorf("invalid status transition from %s to %s", current, next)
	}
	res, err := r.db.ExecContext(ctx, `UPDATE courier_orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`, next, orderID, current)
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
	return r.insertHistoryEntry(ctx, orderID, next, sql.NullString{})
}

func (r *OrdersRepo) insertHistoryEntry(ctx context.Context, orderID int64, status string, note sql.NullString) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO courier_order_status_history (order_id, status, note, created_at) VALUES (?,?,?,CURRENT_TIMESTAMP)`, orderID, status, nullOrString(note))
	return err
}

// CourierStats returns aggregated order counters for a specific courier.
func (r *OrdersRepo) CourierStats(ctx context.Context, courierID int64) (CourierOrderStats, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM courier_orders WHERE courier_id = ? GROUP BY status`, courierID)
	if err != nil {
		return CourierOrderStats{}, err
	}
	defer rows.Close()

	var stats CourierOrderStats
	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return CourierOrderStats{}, err
		}
		stats.Total += count
		if _, ok := activeStatusSet[status]; ok {
			stats.Active += count
		}
		if _, ok := completedStatusSet[status]; ok {
			stats.Completed += count
		}
		if _, ok := canceledStatusSet[status]; ok {
			stats.Canceled += count
		}
	}
	if err := rows.Err(); err != nil {
		return CourierOrderStats{}, err
	}
	return stats, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func statusSet(statuses []string) map[string]struct{} {
	set := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		set[status] = struct{}{}
	}
	return set
}

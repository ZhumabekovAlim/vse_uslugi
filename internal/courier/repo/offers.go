package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const (
	OfferStatusProposed = "proposed"
	OfferStatusAccepted = "accepted"
	OfferStatusDeclined = "declined"
)

// Offer represents a courier pricing proposal.
type Offer struct {
	ID        int64
	OrderID   int64
	CourierID int64
	Price     int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OffersRepo manages courier offer persistence.
type OffersRepo struct {
	db *sql.DB
}

// NewOffersRepo constructs the repository.
func NewOffersRepo(db *sql.DB) *OffersRepo {
	return &OffersRepo{db: db}
}

// Upsert stores or updates a courier price proposal.
func (r *OffersRepo) Upsert(ctx context.Context, orderID, courierID int64, price int) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO courier_offers (order_id, courier_id, price, status) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE price = VALUES(price), status = 'proposed', updated_at = CURRENT_TIMESTAMP`, orderID, courierID, price, "proposed")
	return err
}

// AlreadyOffered returns true if courier already has an offer for the order.
func (r *OffersRepo) AlreadyOffered(ctx context.Context, orderID, courierID int64) (bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT 1 FROM courier_offers WHERE order_id = ? AND courier_id = ?`, orderID, courierID)
	var marker int
	err := row.Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateOffer inserts a lightweight offer stub used by dispatcher to prevent duplicates.
func (r *OffersRepo) CreateOffer(ctx context.Context, orderID, courierID int64, price int) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO courier_offers (order_id, courier_id, price, status) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE updated_at = updated_at`, orderID, courierID, price, OfferStatusProposed)
	return err
}

// UpdateStatus modifies the offer status while keeping the last known price.
func (r *OffersRepo) UpdateStatus(ctx context.Context, orderID, courierID int64, status string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE courier_offers SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE order_id = ? AND courier_id = ?`, status, orderID, courierID)
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

// Get returns a specific offer.
func (r *OffersRepo) Get(ctx context.Context, orderID, courierID int64) (Offer, error) {
	var o Offer
	err := r.db.QueryRowContext(ctx, `SELECT id, order_id, courier_id, price, status, created_at, updated_at FROM courier_offers WHERE order_id = ? AND courier_id = ?`, orderID, courierID).Scan(&o.ID, &o.OrderID, &o.CourierID, &o.Price, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Offer{}, ErrNotFound
	}
	if err != nil {
		return Offer{}, err
	}
	return o, nil
}

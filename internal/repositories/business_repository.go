package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"naimuBack/internal/models"
)

var (
	ErrBusinessAccountSuspended = errors.New("business account suspended")
	ErrNoFreeSeats              = errors.New("no free seats available")
)

type BusinessRepository struct {
	DB *sql.DB
}

// UpsertWorkerListing attaches a listing to a worker within the same business.
func (r *BusinessRepository) UpsertWorkerListing(ctx context.Context, l models.BusinessWorkerListing) error {
	_, err := r.DB.ExecContext(ctx, `
                INSERT INTO business_worker_listings (business_user_id, worker_user_id, listing_type, listing_id)
                VALUES (?, ?, ?, ?)
                ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP
        `, l.BusinessUserID, l.WorkerUserID, l.ListingType, l.ListingID)
	return err
}

// DeleteWorkerListing detaches a listing from a worker.
func (r *BusinessRepository) DeleteWorkerListing(ctx context.Context, l models.BusinessWorkerListing) error {
	_, err := r.DB.ExecContext(ctx, `
                DELETE FROM business_worker_listings
                WHERE business_user_id = ? AND worker_user_id = ? AND listing_type = ? AND listing_id = ?
        `, l.BusinessUserID, l.WorkerUserID, l.ListingType, l.ListingID)
	return err
}

// ListWorkerListings returns all attachments for workers of a business.
func (r *BusinessRepository) ListWorkerListings(ctx context.Context, businessUserID int) (map[int][]models.BusinessWorkerListing, error) {
	rows, err := r.DB.QueryContext(ctx, `
                SELECT worker_user_id, listing_type, listing_id, created_at, updated_at
                FROM business_worker_listings
                WHERE business_user_id = ?
        `, businessUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]models.BusinessWorkerListing)
	for rows.Next() {
		var l models.BusinessWorkerListing
		if err := rows.Scan(&l.WorkerUserID, &l.ListingType, &l.ListingID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.BusinessUserID = businessUserID
		result[l.WorkerUserID] = append(result[l.WorkerUserID], l)
	}
	return result, rows.Err()
}

func (r *BusinessRepository) GetAccountByUserID(ctx context.Context, businessUserID int) (models.BusinessAccount, error) {
	var acc models.BusinessAccount
	query := `SELECT id, business_user_id, seats_total, seats_used, status, created_at, updated_at FROM business_accounts WHERE business_user_id = ?`
	err := r.DB.QueryRowContext(ctx, query, businessUserID).Scan(
		&acc.ID, &acc.BusinessUserID, &acc.SeatsTotal, &acc.SeatsUsed, &acc.Status, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.BusinessAccount{}, nil
		}
		return models.BusinessAccount{}, err
	}
	return acc, nil
}

func (r *BusinessRepository) CreateAccount(ctx context.Context, businessUserID int) (models.BusinessAccount, error) {
	query := `INSERT INTO business_accounts (business_user_id) VALUES (?)`
	res, err := r.DB.ExecContext(ctx, query, businessUserID)
	if err != nil {
		return models.BusinessAccount{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.BusinessAccount{}, err
	}
	return r.GetAccountByUserID(ctx, int(id))
}

func (r *BusinessRepository) AddSeats(ctx context.Context, businessUserID, seats int) error {
	query := `UPDATE business_accounts SET seats_total = seats_total + ?, status = 'active' WHERE business_user_id = ?`
	_, err := r.DB.ExecContext(ctx, query, seats, businessUserID)
	return err
}

func (r *BusinessRepository) IncrementSeatsUsed(ctx context.Context, businessUserID int) error {
	query := `UPDATE business_accounts SET seats_used = seats_used + 1 WHERE business_user_id = ?`
	_, err := r.DB.ExecContext(ctx, query, businessUserID)
	return err
}

func (r *BusinessRepository) SetStatus(ctx context.Context, businessUserID int, status string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE business_accounts SET status = ? WHERE business_user_id = ?`, status, businessUserID)
	return err
}

func (r *BusinessRepository) SaveSeatPurchase(ctx context.Context, purchase models.BusinessSeatPurchase) error {
	query := `INSERT INTO business_seat_purchases (business_user_id, seats, amount, provider, state, provider_txn_id, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?)`
	var payload any
	if purchase.PayloadJSON != nil {
		b, err := json.Marshal(purchase.PayloadJSON)
		if err != nil {
			return err
		}
		payload = b
	}
	_, err := r.DB.ExecContext(ctx, query, purchase.BusinessUserID, purchase.Seats, purchase.Amount, purchase.Provider, purchase.State, purchase.ProviderTxnID, payload)
	return err
}

func (r *BusinessRepository) CreateWorker(ctx context.Context, worker models.BusinessWorker) (models.BusinessWorker, error) {
	query := `INSERT INTO business_workers (business_user_id, worker_user_id, login, chat_id, status, can_respond) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := r.DB.ExecContext(ctx, query, worker.BusinessUserID, worker.WorkerUserID, worker.Login, worker.ChatID, worker.Status, worker.CanRespond)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.BusinessWorker{}, err
	}
	worker.ID = int(id)
	return worker, nil
}

func (r *BusinessRepository) GetWorkerByLogin(ctx context.Context, login string) (models.BusinessWorker, error) {
	var worker models.BusinessWorker
	query := `SELECT id, business_user_id, worker_user_id, login, chat_id, status, can_respond, created_at, updated_at FROM business_workers WHERE login = ?`
	err := r.DB.QueryRowContext(ctx, query, login).Scan(
		&worker.ID, &worker.BusinessUserID, &worker.WorkerUserID, &worker.Login, &worker.ChatID, &worker.Status, &worker.CanRespond, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.BusinessWorker{}, nil
		}
		return models.BusinessWorker{}, err
	}
	return worker, nil
}

func (r *BusinessRepository) GetWorkerByID(ctx context.Context, workerID int) (models.BusinessWorker, error) {
	var worker models.BusinessWorker
	query := `SELECT id, business_user_id, worker_user_id, login, chat_id, status, can_respond, created_at, updated_at FROM business_workers WHERE id = ?`
	err := r.DB.QueryRowContext(ctx, query, workerID).Scan(
		&worker.ID, &worker.BusinessUserID, &worker.WorkerUserID, &worker.Login, &worker.ChatID, &worker.Status, &worker.CanRespond, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.BusinessWorker{}, nil
		}
		return models.BusinessWorker{}, err
	}
	return worker, nil
}

// GetWorkerByUserID fetches a business worker row by underlying user ID.
func (r *BusinessRepository) GetWorkerByUserID(ctx context.Context, workerUserID int) (models.BusinessWorker, error) {
	var worker models.BusinessWorker
	query := `SELECT id, business_user_id, worker_user_id, login, chat_id, status, can_respond, created_at, updated_at FROM business_workers WHERE worker_user_id = ?`
	err := r.DB.QueryRowContext(ctx, query, workerUserID).Scan(
		&worker.ID, &worker.BusinessUserID, &worker.WorkerUserID, &worker.Login, &worker.ChatID, &worker.Status, &worker.CanRespond, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.BusinessWorker{}, nil
		}
		return models.BusinessWorker{}, err
	}
	return worker, nil
}

func (r *BusinessRepository) GetWorkersByBusiness(ctx context.Context, businessUserID int) ([]models.BusinessWorker, error) {
	query := `SELECT bw.id, bw.business_user_id, bw.worker_user_id, bw.login, bw.chat_id, bw.status, bw.can_respond, bw.created_at, bw.updated_at,
       u.name, u.surname, u.phone
FROM business_workers bw
JOIN users u ON u.id = bw.worker_user_id
WHERE bw.business_user_id = ?`
	rows, err := r.DB.QueryContext(ctx, query, businessUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []models.BusinessWorker
	for rows.Next() {
		var worker models.BusinessWorker
		var user models.User
		if err := rows.Scan(&worker.ID, &worker.BusinessUserID, &worker.WorkerUserID, &worker.Login, &worker.ChatID, &worker.Status, &worker.CanRespond, &worker.CreatedAt, &worker.UpdatedAt, &user.Name, &user.Surname, &user.Phone); err != nil {
			return nil, err
		}
		user.ID = worker.WorkerUserID
		worker.User = &user
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

func (r *BusinessRepository) UpdateWorker(ctx context.Context, worker models.BusinessWorker) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE business_workers SET login = ?, status = ?, can_respond = ? WHERE id = ? AND business_user_id = ?`, worker.Login, worker.Status, worker.CanRespond, worker.ID, worker.BusinessUserID)
	return err
}

func (r *BusinessRepository) DeleteWorker(ctx context.Context, worker models.BusinessWorker) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	cleanupQueries := []struct {
		query string
		args  []any
	}{
		{query: `DELETE FROM business_worker_listings WHERE business_user_id = ? AND worker_user_id = ?`, args: []any{worker.BusinessUserID, worker.WorkerUserID}},
		{query: `DELETE FROM messages WHERE chat_id = ?`, args: []any{worker.ChatID}},
		{query: `DELETE FROM chats WHERE id = ?`, args: []any{worker.ChatID}},
		{query: "DELETE FROM courier_offers WHERE courier_id IN (SELECT id FROM couriers WHERE user_id = ?)", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM courier_orders WHERE sender_id = ? OR courier_id IN (SELECT id FROM couriers WHERE user_id = ?)", args: []any{worker.WorkerUserID, worker.WorkerUserID}},
		{query: "DELETE FROM couriers WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM service WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM ad WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM rent WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM rent_ad WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM work WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: "DELETE FROM work_ad WHERE user_id = ?", args: []any{worker.WorkerUserID}},
		{query: `DELETE FROM business_workers WHERE id = ? AND business_user_id = ?`, args: []any{worker.ID, worker.BusinessUserID}},
		{query: `DELETE FROM users WHERE id = ?`, args: []any{worker.WorkerUserID}},
	}

	for _, cleanup := range cleanupQueries {
		if _, err = tx.ExecContext(ctx, cleanup.query, cleanup.args...); err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, `UPDATE business_accounts SET seats_used = CASE WHEN seats_used > 0 THEN seats_used - 1 ELSE 0 END WHERE business_user_id = ?`, worker.BusinessUserID); err != nil {
		return err
	}

	return tx.Commit()
}

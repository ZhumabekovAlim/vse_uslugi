package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"naimuBack/internal/models"
)

type IAPRepository struct {
	DB *sql.DB

	once sync.Once
	err  error
}

func NewIAPRepository(db *sql.DB) *IAPRepository {
	return &IAPRepository{DB: db}
}

func (r *IAPRepository) ensureSchema(ctx context.Context) error {
	r.once.Do(func() {
		const ddl = `
CREATE TABLE IF NOT EXISTS apple_iap_transactions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id VARCHAR(255) NOT NULL,
    original_transaction_id VARCHAR(255) NOT NULL,
    user_id INT NOT NULL,
    product_id VARCHAR(255) DEFAULT '',
    environment VARCHAR(32) DEFAULT '',
    bundle_id VARCHAR(255) DEFAULT '',
    target_json LONGTEXT,
    raw_transaction LONGTEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_transaction_id (transaction_id),
    KEY idx_original_transaction (original_transaction_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
		_, r.err = r.DB.ExecContext(ctx, ddl)
	})
	return r.err
}

// IsProcessed returns true if the transactionId is already stored, ensuring idempotency.
func (r *IAPRepository) IsProcessed(ctx context.Context, transactionID string) (bool, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return false, err
	}
	var exists bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM apple_iap_transactions WHERE transaction_id = ?)`, transactionID).Scan(&exists)
	return exists, err
}

// Save stores the transaction together with the target payload so that renewals can be re-applied.
// It is safe to call multiple times for the same transactionId: duplicates are ignored.
func (r *IAPRepository) Save(ctx context.Context, txn models.AppleTransaction, userID int, target models.IAPTarget) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	if txn.TransactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}
	targetJSON, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("marshal target: %w", err)
	}
	_, err = r.DB.ExecContext(ctx, `
INSERT INTO apple_iap_transactions (transaction_id, original_transaction_id, user_id, product_id, environment, bundle_id, target_json, raw_transaction)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE transaction_id = transaction_id
`, txn.TransactionID, txn.OriginalTransactionID, userID, txn.ProductID, txn.Environment, txn.BundleID, targetJSON, txn.Raw)
	return err
}

// DeleteByTransactionID removes a transaction record. Useful when downstream application fails after saving.
func (r *IAPRepository) DeleteByTransactionID(ctx context.Context, transactionID string) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM apple_iap_transactions WHERE transaction_id = ?`, transactionID)
	return err
}

// FindByOriginalTransactionID returns the latest stored transaction by original transaction id.
func (r *IAPRepository) FindByOriginalTransactionID(ctx context.Context, originalID string) (models.IAPTarget, int, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return models.IAPTarget{}, 0, err
	}
	var (
		targetData string
		userID     int
	)
	err := r.DB.QueryRowContext(ctx, `SELECT target_json, user_id FROM apple_iap_transactions WHERE original_transaction_id = ? ORDER BY id DESC LIMIT 1`, originalID).Scan(&targetData, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.IAPTarget{}, 0, ErrNotFound
		}
		return models.IAPTarget{}, 0, err
	}
	var target models.IAPTarget
	if targetData != "" {
		if unmarshalErr := json.Unmarshal([]byte(targetData), &target); unmarshalErr != nil {
			return models.IAPTarget{}, 0, unmarshalErr
		}
	}
	return target, userID, nil
}

func (r *IAPRepository) GetOwnerByOriginalTransactionID(ctx context.Context, originalID string) (int, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return 0, err
	}

	var userID int
	err := r.DB.QueryRowContext(ctx,
		`SELECT user_id FROM apple_iap_transactions WHERE original_transaction_id = ? ORDER BY id DESC LIMIT 1`,
		originalID,
	).Scan(&userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return userID, nil
}

// ErrNotFound wraps sql.ErrNoRows for clarity.
var ErrNotFound = errors.New("not found")

// GetLatestTransaction returns the most recent transaction row for the given user.
func (r *IAPRepository) GetLatestTransaction(ctx context.Context, userID int) (models.AppleTransaction, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return models.AppleTransaction{}, err
	}
	row := r.DB.QueryRowContext(ctx, `
SELECT transaction_id, original_transaction_id, product_id, environment, bundle_id, raw_transaction, created_at
FROM apple_iap_transactions WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`, userID)

	var txn models.AppleTransaction
	var raw string
	if err := row.Scan(&txn.TransactionID, &txn.OriginalTransactionID, &txn.ProductID, &txn.Environment, &txn.BundleID, &raw, new(time.Time)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.AppleTransaction{}, ErrNotFound
		}
		return models.AppleTransaction{}, err
	}
	txn.Raw = raw
	return txn, nil
}

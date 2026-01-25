package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"naimuBack/internal/models"
)

type GoogleIAPRepository struct {
	DB   *sql.DB
	once sync.Once
	err  error
}

func NewGoogleIAPRepository(db *sql.DB) *GoogleIAPRepository {
	return &GoogleIAPRepository{DB: db}
}

func (r *GoogleIAPRepository) ensureSchema(ctx context.Context) error {
	r.once.Do(func() {
		const ddl = `
CREATE TABLE IF NOT EXISTS google_iap_transactions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  purchase_token VARCHAR(512) NOT NULL,
  order_id VARCHAR(255) DEFAULT '',
  user_id INT NOT NULL,
  product_id VARCHAR(255) DEFAULT '',
  package_name VARCHAR(255) DEFAULT '',
  kind VARCHAR(32) DEFAULT '',
  target_json LONGTEXT,
  raw_purchase LONGTEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_purchase_token (purchase_token),
  KEY idx_order_id (order_id),
  KEY idx_user_id (user_id),
  KEY idx_product_id (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
		_, r.err = r.DB.ExecContext(ctx, ddl)
	})
	return r.err
}

func (r *GoogleIAPRepository) IsProcessed(ctx context.Context, purchaseToken string) (bool, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return false, err
	}
	var exists bool
	err := r.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM google_iap_transactions WHERE purchase_token = ?)`,
		purchaseToken,
	).Scan(&exists)
	return exists, err
}

func (r *GoogleIAPRepository) Save(ctx context.Context, p models.GooglePurchase, userID int, target models.IAPTarget) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	if p.PurchaseToken == "" {
		return fmt.Errorf("purchase_token is required")
	}

	targetJSON, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("marshal target: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, `
INSERT INTO google_iap_transactions (purchase_token, order_id, user_id, product_id, package_name, kind, target_json, raw_purchase)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE purchase_token = purchase_token
`,
		p.PurchaseToken,
		p.OrderID,
		userID,
		p.ProductID,
		p.PackageName,
		p.Kind,
		targetJSON,
		p.Raw,
	)
	return err
}

func (r *GoogleIAPRepository) DeleteByToken(ctx context.Context, purchaseToken string) error {
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM google_iap_transactions WHERE purchase_token = ?`, purchaseToken)
	return err
}

// Anti-theft: один purchase_token должен принадлежать одному user.
func (r *GoogleIAPRepository) GetOwnerByToken(ctx context.Context, purchaseToken string) (int, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return 0, err
	}
	var userID int
	err := r.DB.QueryRowContext(ctx,
		`SELECT user_id FROM google_iap_transactions WHERE purchase_token = ? ORDER BY id DESC LIMIT 1`,
		purchaseToken,
	).Scan(&userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (r *GoogleIAPRepository) FindTargetByToken(ctx context.Context, purchaseToken string) (models.IAPTarget, int, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return models.IAPTarget{}, 0, err
	}

	var (
		targetData string
		userID     int
	)
	err := r.DB.QueryRowContext(ctx,
		`SELECT target_json, user_id FROM google_iap_transactions WHERE purchase_token = ? ORDER BY id DESC LIMIT 1`,
		purchaseToken,
	).Scan(&targetData, &userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.IAPTarget{}, 0, ErrNotFound
		}
		return models.IAPTarget{}, 0, err
	}

	var target models.IAPTarget
	if targetData != "" {
		if uerr := json.Unmarshal([]byte(targetData), &target); uerr != nil {
			return models.IAPTarget{}, 0, uerr
		}
	}
	return target, userID, nil
}

func (r *GoogleIAPRepository) ListByUser(ctx context.Context, userID int) ([]models.GoogleIAPHistory, error) {
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT purchase_token, order_id, product_id, package_name, kind, target_json, created_at
FROM google_iap_transactions
WHERE user_id = ?
ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.GoogleIAPHistory
	for rows.Next() {
		var item models.GoogleIAPHistory
		var target sql.NullString
		if err := rows.Scan(&item.PurchaseToken, &item.OrderID, &item.ProductID, &item.PackageName, &item.Kind, &target, &item.CreatedAt); err != nil {
			return nil, err
		}
		if target.Valid {
			item.TargetJSON = json.RawMessage(target.String)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

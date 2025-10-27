package repo

import (
    "context"
    "database/sql"
)

// PaymentsRepo handles payments tables.
type PaymentsRepo struct {
    db *sql.DB
}

// NewPaymentsRepo creates repo.
func NewPaymentsRepo(db *sql.DB) *PaymentsRepo { return &PaymentsRepo{db: db} }

// Create inserts payment record.
func (r *PaymentsRepo) Create(ctx context.Context, orderID int64, amount int, provider string, payload []byte) (int64, error) {
    res, err := r.db.ExecContext(ctx, `INSERT INTO payments (order_id, amount, provider, payload_json) VALUES (?,?,?,?)`, orderID, amount, provider, payload)
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

// UpdateState updates payment state and provider transaction ID.
func (r *PaymentsRepo) UpdateState(ctx context.Context, paymentID int64, state, providerTxn string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE payments SET state = ?, provider_txn_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, state, providerTxn, paymentID)
    return err
}

// UpdateStateByOrder updates payment by order id.
func (r *PaymentsRepo) UpdateStateByOrder(ctx context.Context, orderID int64, state, providerTxn string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE payments SET state = ?, provider_txn_id = ?, updated_at = CURRENT_TIMESTAMP WHERE order_id = ?`, state, providerTxn, orderID)
    return err
}

// SaveWebhook stores webhook payload.
func (r *PaymentsRepo) SaveWebhook(ctx context.Context, provider, signature string, payload []byte) error {
    _, err := r.db.ExecContext(ctx, `INSERT INTO payment_webhooks (provider, signature, body_json) VALUES (?,?,?)`, provider, signature, payload)
    return err
}

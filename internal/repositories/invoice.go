package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"naimuBack/internal/models"
)

type InvoiceRepo struct{ DB *sql.DB }

func NewInvoiceRepo(db *sql.DB) *InvoiceRepo { return &InvoiceRepo{DB: db} }

func (r *InvoiceRepo) CreateInvoice(ctx context.Context, userID int, amount float64, description string) (int, error) {
	const q = `INSERT INTO invoices (user_id, amount, description, status) VALUES (?, ?, ?, 'pending')`
	res, err := r.DB.ExecContext(ctx, q, userID, amount, description)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *InvoiceRepo) MarkPaid(ctx context.Context, invID int) error {
	return r.UpdateStatus(ctx, invID, "paid")
}

func (r *InvoiceRepo) UpdateStatus(ctx context.Context, invID int, status string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE invoices SET status=? WHERE inv_id=?`, status, invID)
	return err
}

func (r *InvoiceRepo) GetByID(ctx context.Context, invID int) (models.Invoice, error) {
	const q = `SELECT inv_id, user_id, amount, description, status, created_at FROM invoices WHERE inv_id = ?`
	var inv models.Invoice
	err := r.DB.QueryRowContext(ctx, q, invID).Scan(&inv.ID, &inv.UserID, &inv.Amount, &inv.Description, &inv.Status, &inv.CreatedAt)
	if err != nil {
		return models.Invoice{}, err
	}
	return inv, nil
}

func (r *InvoiceRepo) GetByUser(ctx context.Context, userID int) ([]models.Invoice, error) {
	const q = `SELECT inv_id, user_id, amount, description, status, created_at FROM invoices WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.ID, &inv.UserID, &inv.Amount, &inv.Description, &inv.Status, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// AddTarget stores metadata describing an action to execute once the invoice is paid.
func (r *InvoiceRepo) AddTarget(ctx context.Context, invoiceID int, targetType string, targetID int64, payload json.RawMessage) (int, error) {
	res, err := r.DB.ExecContext(ctx, `INSERT INTO invoice_targets (invoice_id, target_type, target_id, payload_json) VALUES (?,?,?,?)`, invoiceID, targetType, targetID, payload)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// ListTargets fetches all actions associated with the invoice.
func (r *InvoiceRepo) ListTargets(ctx context.Context, invoiceID int) ([]models.InvoiceTarget, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, invoice_id, target_type, target_id, payload_json, processed_at, created_at FROM invoice_targets WHERE invoice_id = ?`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.InvoiceTarget
	for rows.Next() {
		var target models.InvoiceTarget
		var processed sql.NullTime
		if err := rows.Scan(&target.ID, &target.InvoiceID, &target.TargetType, &target.TargetID, &target.Payload, &processed, &target.CreatedAt); err != nil {
			return nil, err
		}
		if processed.Valid {
			t := processed.Time
			target.ProcessedAt = &t
		}
		out = append(out, target)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkTargetProcessed marks the invoice target as processed to avoid duplicate executions.
func (r *InvoiceRepo) MarkTargetProcessed(ctx context.Context, targetID int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE invoice_targets SET processed_at = ? WHERE id = ?`, time.Now(), targetID)
	return err
}

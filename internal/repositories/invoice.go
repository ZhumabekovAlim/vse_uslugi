package repositories

import (
	"context"
	"database/sql"

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

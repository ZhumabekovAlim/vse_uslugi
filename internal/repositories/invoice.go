package repositories

import (
	"context"
	"database/sql"
)

type InvoiceRepo struct{ DB *sql.DB }

func NewInvoiceRepo(db *sql.DB) *InvoiceRepo { return &InvoiceRepo{DB: db} }

func (r *InvoiceRepo) CreateInvoice(ctx context.Context, amount float64, description string) (int, error) {
	const q = `INSERT INTO invoices (amount, description, status) VALUES (?, ?, 'pending')`
	res, err := r.DB.ExecContext(ctx, q, amount, description)
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
	_, err := r.DB.ExecContext(ctx, `UPDATE invoices SET status='paid' WHERE inv_id=?`, invID)
	return err
}

package models

import (
	"encoding/json"
	"time"
)

// Invoice represents a payment invoice created for a user.
type Invoice struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// InvoiceTarget describes a follow-up action associated with the invoice payment.
type InvoiceTarget struct {
	ID          int             `json:"id"`
	InvoiceID   int             `json:"invoice_id"`
	TargetType  string          `json:"target_type"`
	TargetID    int64           `json:"target_id"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

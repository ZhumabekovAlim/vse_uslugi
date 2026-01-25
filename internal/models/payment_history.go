package models

import (
	"encoding/json"
	"time"
)

type PaymentHistoryItem struct {
	Provider          string            `json:"provider"`
	CreatedAt         time.Time         `json:"created_at"`
	Invoice           *Invoice          `json:"invoice,omitempty"`
	AppleTransaction  *AppleIAPHistory  `json:"apple_transaction,omitempty"`
	GoogleTransaction *GoogleIAPHistory `json:"google_transaction,omitempty"`
}

type AppleIAPHistory struct {
	TransactionID         string          `json:"transaction_id"`
	OriginalTransactionID string          `json:"original_transaction_id"`
	ProductID             string          `json:"product_id"`
	Environment           string          `json:"environment"`
	BundleID              string          `json:"bundle_id"`
	TargetJSON            json.RawMessage `json:"target_json,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

type GoogleIAPHistory struct {
	PurchaseToken string          `json:"purchase_token"`
	OrderID       string          `json:"order_id"`
	ProductID     string          `json:"product_id"`
	PackageName   string          `json:"package_name"`
	Kind          string          `json:"kind"`
	TargetJSON    json.RawMessage `json:"target_json,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

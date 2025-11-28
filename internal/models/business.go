package models

import "time"

// BusinessAccount describes aggregated seat information for a business user.
type BusinessAccount struct {
	ID             int        `json:"id"`
	BusinessUserID int        `json:"business_user_id"`
	SeatsTotal     int        `json:"seats_total"`
	SeatsUsed      int        `json:"seats_used"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// BusinessWorker represents a worker managed by a business account.
type BusinessWorker struct {
	ID             int        `json:"id"`
	BusinessUserID int        `json:"business_user_id"`
	WorkerUserID   int        `json:"worker_user_id"`
	Login          string     `json:"login"`
	ChatID         int        `json:"chat_id"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`

	// Embedded user data for listing pages
	User *User `json:"user,omitempty"`
}

// BusinessSeatPurchase captures a seat purchase event.
type BusinessSeatPurchase struct {
	ID             int        `json:"id"`
	BusinessUserID int        `json:"business_user_id"`
	Seats          int        `json:"seats"`
	Amount         float64    `json:"amount"`
	Provider       *string    `json:"provider,omitempty"`
	State          *string    `json:"state,omitempty"`
	ProviderTxnID  *string    `json:"provider_txn_id,omitempty"`
	PayloadJSON    any        `json:"payload_json,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// BusinessWorkerListing links listings to workers.
type BusinessWorkerListing struct {
	ID             int        `json:"id"`
	BusinessUserID int        `json:"business_user_id"`
	WorkerUserID   int        `json:"worker_user_id"`
	ListingType    string     `json:"listing_type"`
	ListingID      int        `json:"listing_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

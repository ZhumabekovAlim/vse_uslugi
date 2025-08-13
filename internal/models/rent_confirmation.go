package models

import "time"

type RentConfirmation struct {
	ID          int        `json:"id"`
	RentID      int        `json:"rent_id"`
	ChatID      int        `json:"chat_id"`
	ClientID    int        `json:"client_id"`
	PerformerID int        `json:"performer_id"`
	Confirmed   bool       `json:"confirmed"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

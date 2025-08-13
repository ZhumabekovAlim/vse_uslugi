package models

import "time"

type AdConfirmation struct {
	ID          int        `json:"id"`
	AdID        int        `json:"ad_id"`
	ChatID      int        `json:"chat_id"`
	ClientID    int        `json:"client_id"`
	PerformerID int        `json:"performer_id"`
	Confirmed   bool       `json:"confirmed"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

package models

import "time"

type WorkConfirmation struct {
	ID          int        `json:"id"`
	WorkID      int        `json:"work_id"`
	ChatID      int        `json:"chat_id"`
	ClientID    int        `json:"client_id"`
	PerformerID int        `json:"performer_id"`
	Confirmed   bool       `json:"confirmed"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

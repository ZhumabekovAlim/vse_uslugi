package models

import (
	"time"
)

type RentAdResponses struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id,omitempty"`
	RentAdID    int        `json:"rent_ad_id,omitempty"`
	ChatID      int        `json:"chat_id,omitempty"`
	ClientID    int        `json:"client_id,omitempty"`
	PerformerID int        `json:"performer_id,omitempty"`
	Price       float64    `json:"price"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

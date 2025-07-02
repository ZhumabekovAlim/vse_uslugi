package models

import (
	"time"
)

type AdResponses struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id,omitempty"`
	AdID        int        `json:"ad_id,omitempty"`
	Price       float64    `json:"price"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

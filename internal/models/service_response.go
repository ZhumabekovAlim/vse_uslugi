package models

import (
	"time"
)

type ServiceResponses struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id,omitempty"`
	ServiceID   int        `json:"ad_id,omitempty"`
	Price       float64    `json:"price"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

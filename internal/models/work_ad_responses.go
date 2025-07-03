package models

import (
	"time"
)

type WorkAdResponses struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id,omitempty"`
	WorkAdID    int        `json:"work_ad_id,omitempty"`
	Price       float64    `json:"price"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

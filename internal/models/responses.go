package models

import (
	"time"
)

type Responses struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	AdID      int        `json:"ad_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

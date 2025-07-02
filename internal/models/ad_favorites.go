package models

import (
	"time"
)

type AdFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	AdID      int        `json:"ad_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

package models

import (
	"time"
)

type RentFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	RentID    int        `json:"rent_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

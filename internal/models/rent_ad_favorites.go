package models

import (
	"time"
)

type RentAdFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	RentAdID  int        `json:"rent_ad_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

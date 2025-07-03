package models

import (
	"time"
)

type WorkAdFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	WorkAdID  int        `json:"work_ad_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

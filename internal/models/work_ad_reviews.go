package models

import (
	"time"
)

type WorkAdReviews struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	WorkAdID  int        `json:"work_ad_id,omitempty"`
	Rating    float64    `json:"rating"`
	Review    string     `json:"review"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

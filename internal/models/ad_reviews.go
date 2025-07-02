package models

import (
	"time"
)

type AdReviews struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	AdID      int        `json:"ad_id,omitempty"`
	Rating    float64    `json:"rating"`
	Review    string     `json:"review"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

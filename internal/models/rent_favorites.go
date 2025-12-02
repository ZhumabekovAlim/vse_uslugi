package models

import (
	"time"
)

type RentFavorite struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	RentID       int       `json:"rent_id"`
	Name         string    `json:"name"`
	Price        *float64  `json:"price"`
	PriceTo      *float64  `json:"price_to"`
	WorkTimeFrom string    `json:"work_time_from"`
	WorkTimeTo   string    `json:"work_time_to"`
	Negotiable   bool      `json:"negotiable"`
	HidePhone    bool      `json:"hide_phone"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

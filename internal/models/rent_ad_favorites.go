package models

import (
	"time"
)

type RentAdFavorite struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	RentAdID     int       `json:"rent_ad_id"`
	Name         string    `json:"name"`
	Price        *float64  `json:"price"`
	PriceTo      *float64  `json:"price_to"`
	WorkTimeFrom string    `json:"work_time_from"`
	WorkTimeTo   string    `json:"work_time_to"`
	Negotiable   bool      `json:"negotiable"`
	HidePhone    bool      `json:"hide_phone"`
	ImagePath    *string   `json:"image_path,omitempty"`
	OrderDate    *string   `json:"order_date,omitempty"`
	OrderTime    *string   `json:"order_time,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

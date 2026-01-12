package models

import (
	"time"
)

type WorkFavorite struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	WorkID     int       `json:"work_id"`
	CityID     int       `json:"city_id"`
	Name       string    `json:"name"`
	Price      *float64  `json:"price"`
	PriceTo    *float64  `json:"price_to"`
	Negotiable bool      `json:"negotiable"`
	HidePhone  bool      `json:"hide_phone"`
	ImagePath  *string   `json:"image_path,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

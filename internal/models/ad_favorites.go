package models

import (
	"time"
)

type AdFavorite struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	AdID       int       `json:"ad_id"`
	Name       string    `json:"name"`
	Price      *float64  `json:"price"`
	PriceTo    *float64  `json:"price_to"`
	OnSite     bool      `json:"on_site"`
	Negotiable bool      `json:"negotiable"`
	HidePhone  bool      `json:"hide_phone"`
	ImagePath  *string   `json:"image_path,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

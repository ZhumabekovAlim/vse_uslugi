package models

import (
	"time"
)

type ServiceFavorite struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ServiceID  int       `json:"service_id"`
	CityID     int       `json:"city_id"`
	CityName   string    `json:"city_name"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	Price      *float64  `json:"price"`
	PriceTo    *float64  `json:"price_to"`
	OnSite     bool      `json:"on_site"`
	Negotiable bool      `json:"negotiable"`
	HidePhone  bool      `json:"hide_phone"`
	ImagePath  *string   `json:"image_path,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

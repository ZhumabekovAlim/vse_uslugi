package models

import (
	"time"
)

type AdFavorite struct {
	ID         int      `json:"id"`
	UserID     int      `json:"user_id"`
	AdID       int      `json:"ad_id"`
	CityID     int      `json:"city_id"`
	CityName   string   `json:"city_name"`
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	Price      *float64 `json:"price"`
	PriceTo    *float64 `json:"price_to"`
	OnSite     bool     `json:"on_site"`
	Negotiable bool     `json:"negotiable"`
	HidePhone  bool     `json:"hide_phone"`
	User       struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Surname      string  `json:"surname"`
		Phone        string  `json:"phone"`
		ReviewRating float64 `json:"review_rating"`
		ReviewsCount int     `json:"reviews_count"`
		AvatarPath   *string `json:"avatar_path,omitempty"`
	} `json:"user"`
	ImagePath *string   `json:"image_path,omitempty"`
	OrderDate *string   `json:"order_date,omitempty"`
	OrderTime *string   `json:"order_time,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

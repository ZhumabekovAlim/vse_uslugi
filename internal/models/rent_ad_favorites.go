package models

import (
	"time"
)

type RentAdFavorite struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RentAdID  int       `json:"rent_ad_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

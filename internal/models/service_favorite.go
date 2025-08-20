package models

import (
	"time"
)

type ServiceFavorite struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ServiceID int       `json:"service_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

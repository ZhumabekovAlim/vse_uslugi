package models

import (
	"time"
)

type WorkFavorite struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	WorkID    int       `json:"work_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

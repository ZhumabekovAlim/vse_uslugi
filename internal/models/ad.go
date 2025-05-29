package models

import (
	"time"
)

type Ad struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Address     string     `json:"address"`
	Price       float64    `json:"price"`
	UserID      int        `json:"user_id, omitempty"`
	Images      string     `json:"images"`
	CategoryID  int        `json:"category_id, omitempty"`
	Description string     `json:"description"`
	AvgRating   float64    `json:"avg_rating"`
	ReviewsID   int        `json:"reviews_id, omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

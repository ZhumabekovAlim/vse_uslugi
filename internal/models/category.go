package models

import (
	"time"
)

type Category struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	ImagePath      string    `json:"image_path"`
	SubcategoryIDs []int     `json:"subcategory_id, omitempty"`
	MinPrice       float64   `json:"min_price"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

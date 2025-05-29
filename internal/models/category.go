package models

import (
	"time"
)

type Category struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	ImagePath     string     `json:"image_path"`
	Subcategories string     `json:"subcategories"`
	MinPrice      float64    `json:"min_price"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

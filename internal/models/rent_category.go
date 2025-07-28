package models

import "time"

type RentCategory struct {
	ID             int               `json:"id"`
	Name           string            `json:"name"`
	ImagePath      string            `json:"image_path"`
	Subcategories  []RentSubcategory `json:"subcategories,omitempty"`
	MinPrice       float64           `json:"min_price"`
	SubcategoryIDs []int             `json:"subcategory_id,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
}

package models

import "time"

type Subcategory struct {
	ID         int        `json:"id"`
	CategoryID int        `json:"category_id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

package models

import "time"

type WorkSubcategory struct {
	ID         int        `json:"id"`
	CategoryID int        `json:"category_id"`
	Name       string     `json:"name"`
	NameKz     string     `json:"name_kz"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

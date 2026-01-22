package models

import "time"

// UserItem represents a generic user-owned item across different entities.
type UserItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	CityName    string    `json:"city_name"`
	Price       *float64  `json:"price"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
}

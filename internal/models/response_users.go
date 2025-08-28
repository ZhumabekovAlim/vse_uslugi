package models

import "time"

// ResponseUser represents a user who responded to an item along with response details.
type ResponseUser struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Surname     string    `json:"surname"`
	AvatarPath  *string   `json:"avatar_path,omitempty"`
	Rating      float64   `json:"rating"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

package models

// Location represents a user's geographic coordinates.
type Location struct {
	UserID    int     `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

package models

import "time"

type AdComplaint struct {
	ID          int       `json:"id"`
	AdID        int       `json:"ad_id"`
	UserID      int       `json:"user_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

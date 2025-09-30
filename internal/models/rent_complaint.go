package models

import "time"

type RentComplaint struct {
	ID          int           `json:"id"`
	RentID      int           `json:"rent_id"`
	UserID      int           `json:"user_id"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	User        ComplaintUser `json:"user"`
}

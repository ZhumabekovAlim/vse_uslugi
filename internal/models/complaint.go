package models

import "time"

type Complaint struct {
	ID          int           `json:"id"`
	ServiceID   int           `json:"service_id"`
	UserID      int           `json:"user_id"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	User        ComplaintUser `json:"user"`
}

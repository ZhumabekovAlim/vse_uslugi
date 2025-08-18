package models

import "time"

type WorkComplaint struct {
	ID          int       `json:"id"`
	WorkID      int       `json:"work_id"`
	UserID      int       `json:"user_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

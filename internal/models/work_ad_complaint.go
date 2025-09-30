package models

import "time"

type WorkAdComplaint struct {
	ID          int           `json:"id"`
	WorkAdID    int           `json:"work_ad_id"`
	UserID      int           `json:"user_id"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	User        ComplaintUser `json:"user"`
}

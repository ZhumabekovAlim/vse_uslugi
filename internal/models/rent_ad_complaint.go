package models

import "time"

type RentAdComplaint struct {
	ID          int       `json:"id"`
	RentAdID    int       `json:"rent_ad_id"`
	UserID      int       `json:"user_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

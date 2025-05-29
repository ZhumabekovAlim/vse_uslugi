package models

import (
	"time"
)

type ServiceFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	ServiceID int        `json:"service_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

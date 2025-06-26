package models

import (
	"time"
)

type WorkFavorite struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	WorkID    int        `json:"work_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

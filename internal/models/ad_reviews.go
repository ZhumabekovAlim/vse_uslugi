package models

import (
	"time"
)

type AdReviews struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id,omitempty"`
	AdID           int        `json:"ad_id,omitempty"`
	Rating         float64    `json:"rating"`
	Review         string     `json:"review"`
	UserName       *string    `json:"user_name,omitempty"`
	UserSurname    *string    `json:"user_surname,omitempty"`
	UserAvatarPath *string    `json:"user_avatar_path,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

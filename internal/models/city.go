package models

import (
	"time"
)

type City struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	ParentID  *int       `json:"parent_id,omitempty"`
	Cities    []City     `json:"cities,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

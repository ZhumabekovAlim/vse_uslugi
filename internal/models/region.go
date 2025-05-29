package models

import (
	"time"
)

type Region struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Cities    string     `json:"cities"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

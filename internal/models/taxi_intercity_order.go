package models

import "time"

type TaxiIntercityOrder struct {
	ID            int        `json:"id"`
	ClientID      int        `json:"client_id"`
	ClientName    string     `json:"client_name"`
	ClientPhone   string     `json:"client_phone"`
	FromCity      string     `json:"from_city"`
	ToCity        string     `json:"to_city"`
	TripType      string     `json:"trip_type"`
	Comment       string     `json:"comment,omitempty"`
	Price         float64    `json:"price"`
	DepartureDate time.Time  `json:"departure_date"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
}

type CreateTaxiIntercityOrderRequest struct {
	FromCity      string  `json:"from_city"`
	ToCity        string  `json:"to_city"`
	TripType      string  `json:"trip_type"`
	Comment       string  `json:"comment"`
	Price         float64 `json:"price"`
	DepartureDate string  `json:"departure_date"`
}

type TaxiIntercityOrderFilter struct {
	FromCity      string
	ToCity        string
	DepartureDate *time.Time
	Status        string
	Limit         int
	Offset        int
}

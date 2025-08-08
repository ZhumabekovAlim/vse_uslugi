package models

import "time"

// UserResponseItem represents a single response with the associated item details.
type UserResponseItem struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Description   string    `json:"description"`
	ResponsePrice float64   `json:"response_price"`
	ResponseDate  time.Time `json:"response_date"`
	Type          string    `json:"type"`
}

// UserResponses aggregates responses across different entity types.
type UserResponses struct {
	Service []UserResponseItem `json:"service"`
	Ad      []UserResponseItem `json:"ad"`
	Work    []UserResponseItem `json:"work"`
	WorkAd  []UserResponseItem `json:"work_ad"`
	Rent    []UserResponseItem `json:"rent"`
	RentAd  []UserResponseItem `json:"rent_ad"`
}

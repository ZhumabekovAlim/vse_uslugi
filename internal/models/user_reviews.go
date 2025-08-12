package models

import "time"

// UserReviewItem represents a single review with the associated item details.
type UserReviewItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	Rating      float64   `json:"rating"`
	Review      string    `json:"review"`
	ReviewDate  time.Time `json:"review_date"`
	Type        string    `json:"type"`
}

// UserReviews aggregates reviews across different entity types.
type UserReviews struct {
	Service []UserReviewItem `json:"service"`
	Ad      []UserReviewItem `json:"ad"`
	Work    []UserReviewItem `json:"work"`
	WorkAd  []UserReviewItem `json:"work_ad"`
	Rent    []UserReviewItem `json:"rent"`
	RentAd  []UserReviewItem `json:"rent_ad"`
}

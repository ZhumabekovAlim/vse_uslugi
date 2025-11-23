package models

import "time"

// ResponseUser represents a user who responded to an item along with response details.
type ResponseUser struct {
	ID            int                 `json:"id"`
	Name          string              `json:"name"`
	Surname       string              `json:"surname"`
	AvatarPath    *string             `json:"avatar_path,omitempty"`
	ReviewRating  float64             `json:"review_rating"`
	ReviewsCount  int                 `json:"reviews_count"`
	Phone         string              `json:"phone"`
	Price         float64             `json:"price"`
	Description   string              `json:"description"`
	CreatedAt     time.Time           `json:"created_at"`
	Status        string              `json:"status"`
	ChatID        int                 `json:"chat_id"`
	LastMessage   string              `json:"lastMessage"`
	ProviderPhone string              `json:"provider_phone"`
	ClientPhone   string              `json:"client_phone"`
	MyRole        string              `json:"my_role"`
	ServiceReview *ResponseUserReview `json:"service_review,omitempty"`
	AdReview      *ResponseUserReview `json:"ad_review,omitempty"`
	RentReview    *ResponseUserReview `json:"rent_review,omitempty"`
	WorkReview    *ResponseUserReview `json:"work_review,omitempty"`
	RentAdReview  *ResponseUserReview `json:"rent_ad_review,omitempty"`
	WorkAdReview  *ResponseUserReview `json:"work_ad_review,omitempty"`
}

// ResponseUserReview represents a review left for a specific item by a user.
type ResponseUserReview struct {
	UserID int     `json:"user_id"`
	Rating float64 `json:"rating"`
	Review string  `json:"review"`
}

// ItemResponse aggregates item data with responders.
type ItemResponse struct {
	ItemType    string
	ItemID      int
	ItemName    string
	PerformerID *int
	Status      string
	Users       []ResponseUser
}

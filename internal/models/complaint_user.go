package models

// ComplaintUser contains public user information associated with complaints.
type ComplaintUser struct {
	Name         string  `json:"name"`
	Surname      string  `json:"surname"`
	Email        string  `json:"email"`
	CityID       *int    `json:"city_id,omitempty"`
	AvatarPath   *string `json:"avatar_path,omitempty"`
	ReviewRating float64 `json:"review_rating"`
}

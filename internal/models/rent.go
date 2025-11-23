package models

import (
	"time"
)

type Rent struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Price   float64 `json:"price"`
	UserID  int     `json:"user_id"`
	User    struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Surname      string  `json:"surname"`
		Phone        string  `json:"phone"`
		ReviewRating float64 `json:"review_rating"`
		ReviewsCount int     `json:"reviews_count"`
		AvatarPath   *string `json:"avatar_path,omitempty"`
	} `json:"user"`
	Images            []ImageRent    `json:"images"`
	Videos            []Video        `json:"videos"`
	CategoryID        int            `json:"category_id, omitempty"`
	SubcategoryID     int            `json:"subcategory_id, omitempty"`
	Description       string         `json:"description"`
	AvgRating         float64        `json:"avg_rating"`
	Top               string         `json:"top, omitempty"`
	TopActive         bool           `json:"is_top"`
	TopExpiresAt      *time.Time     `json:"top_expires_at,omitempty"`
	Liked             bool           `json:"liked, omitempty"`
	Responded         bool           `json:"is_responded"`
	ResponseUsers     []ResponseUser `json:"response_users,omitempty"`
	Status            string         `json:"status, omitempty"`
	CategoryName      string         `json:"category_name"`
	SubcategoryName   string         `json:"subcategory_name"`
	SubcategoryNameKz string         `json:"subcategory_name_kz"`
	RentType          string         `json:"rent_type"`
	Deposit           string         `json:"deposit"`
	Latitude          string         `json:"latitude,omitempty"`
	Longitude         string         `json:"longitude,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         *time.Time     `json:"updated_at,omitempty"`
}

type ImageRent struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type RentFilterRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	SortOption    int       `json:"sort"`
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
}

type RentListResponse struct {
	Rents    []Rent  `json:"rents"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

type FilterRentRequest struct {
	CategoryIDs    []int   `json:"category_id"`
	SubcategoryIDs []int   `json:"subcategory_id"`
	PriceFrom      float64 `json:"price_from"`
	PriceTo        float64 `json:"price_to"`
	AvgRatings     []int   `json:"avg_rating"`
	Sorting        int     `json:"sorting"` // 1 - by reviews, 2 - price desc, 3 - price asc
	UserID         int     `json:"user_id,omitempty"`
	CityID         int     `json:"city_id,omitempty"`
}

type FilteredRent struct {
	UserID           int         `json:"user_id"`
	UserName         string      `json:"user_name"`
	UserSurname      string      `json:"user_surname"`
	UserPhone        string      `json:"-"`
	UserAvatarPath   *string     `json:"user_avatar_path,omitempty"`
	UserRating       float64     `json:"user_rating"`
	UserReviewsCount int         `json:"user_reviews_count"`
	RentID           int         `json:"rent_id"`
	RentName         string      `json:"rent_name"`
	RentAddress      string      `json:"rent_address"`
	RentPrice        float64     `json:"rent_price"`
	RentDescription  string      `json:"rent_description"`
	Images           []ImageRent `json:"images"`
	Videos           []Video     `json:"videos"`
	RentLatitude     string      `json:"latitude"`
	RentLongitude    string      `json:"longitude"`
	Liked            bool        `json:"liked"`
	Responded        bool        `json:"is_responded"`
	Top              string      `json:"-"`
	CreatedAt        time.Time   `json:"-"`
}

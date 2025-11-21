package models

import (
	"time"
)

type Work struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Price   float64 `json:"price"`
	UserID  int     `json:"user_id"`
	User    struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Surname      string  `json:"surname"`
		Phone        string  `json:"phone,omitempty"`
		ReviewRating float64 `json:"review_rating"`
		ReviewsCount int     `json:"reviews_count"`
		AvatarPath   *string `json:"avatar_path,omitempty"`
	} `json:"user"`
	Images            []ImageWork `json:"images"`
	Videos            []Video     `json:"videos"`
	CategoryID        int         `json:"category_id, omitempty"`
	SubcategoryID     int         `json:"subcategory_id, omitempty"`
	Description       string      `json:"description"`
	AvgRating         float64     `json:"avg_rating"`
	Top               string      `json:"top, omitempty"`
	Liked             bool        `json:"liked, omitempty"`
	Responded         bool        `json:"is_responded"`
	Status            string      `json:"status, omitempty"`
	CategoryName      string      `json:"category_name"`
	SubcategoryName   string      `json:"subcategory_name"`
	SubcategoryNameKz string      `json:"subcategory_name_kz"`
	WorkExperience    string      `json:"work_experience,omitempty"`
	CityID            int         `json:"city_id"`
	CityName          string      `json:"city_name"`
	CityType          string      `json:"city_type"`
	Schedule          string      `json:"schedule, omitempty"`
	DistanceWork      string      `json:"distance_work,omitempty"`
	PaymentPeriod     string      `json:"payment_period,omitempty"`
	Latitude          string      `json:"latitude,omitempty"`
	Longitude         string      `json:"longitude,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         *time.Time  `json:"updated_at,omitempty"`
}

type ImageWork struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type WorkFilterRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	SortOption    int       `json:"sort"`
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
}

type WorkListResponse struct {
	Works    []Work  `json:"works"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

type FilterWorkRequest struct {
	CategoryIDs    []int   `json:"category_id"`
	SubcategoryIDs []int   `json:"subcategory_id"`
	PriceFrom      float64 `json:"price_from"`
	PriceTo        float64 `json:"price_to"`
	AvgRatings     []int   `json:"avg_rating"`
	Sorting        int     `json:"sorting"` // 1 - by reviews, 2 - price desc, 3 - price asc
	UserID         int     `json:"user_id,omitempty"`
	CityID         int     `json:"city_id,omitempty"`
}

type FilteredWork struct {
	UserID           int         `json:"user_id"`
	UserName         string      `json:"user_name"`
	UserSurname      string      `json:"user_surname"`
	UserPhone        string      `json:"-"`
	UserAvatarPath   *string     `json:"user_avatar_path,omitempty"`
	UserRating       float64     `json:"user_rating"`
	UserReviewsCount int         `json:"user_reviews_count"`
	WorkID           int         `json:"work_id"`
	WorkName         string      `json:"work_name"`
	WorkAddress      string      `json:"work_address"`
	WorkPrice        float64     `json:"work_price"`
	WorkDescription  string      `json:"work_description"`
	Images           []ImageWork `json:"images"`
	Videos           []Video     `json:"videos"`
	WorkLatitude     string      `json:"latitude"`
	WorkLongitude    string      `json:"longitude"`
	Liked            bool        `json:"liked"`
	Responded        bool        `json:"is_responded"`
	Top              string      `json:"-"`
	CreatedAt        time.Time   `json:"-"`
}

package models

import (
	"time"
)

type Service struct {
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
	Images          []Image    `json:"images"`
	CategoryID      int        `json:"category_id, omitempty"`
	SubcategoryID   int        `json:"subcategory_id, omitempty"`
	Description     string     `json:"description"`
	AvgRating       float64    `json:"avg_rating"`
	Top             string     `json:"top, omitempty"`
	Liked           bool       `json:"liked, omitempty"`
	Responded       bool       `json:"is_responded"`
	Status          string     `json:"status, omitempty"`
	CategoryName    string     `json:"category_name"`
	SubcategoryName string     `json:"subcategory_name"`
	Latitude        *string    `json:"latitude,omitempty"`
	Longitude       *string    `json:"longitude,omitempty"`
	MainCategory    string     `json:"main_category"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type Image struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type ServiceFilterRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	SortOption    int       `json:"sort"`
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
}

type ServiceListResponse struct {
	Services []Service `json:"services"`
	MinPrice float64   `json:"min_price"`
	MaxPrice float64   `json:"max_price"`
}

type FilterServicesRequest struct {
	CategoryIDs    []int   `json:"category_id"`
	SubcategoryIDs []int   `json:"subcategory_id"`
	PriceFrom      float64 `json:"price_from"`
	PriceTo        float64 `json:"price_to"`
	AvgRatings     []int   `json:"avg_rating"`
	Sorting        int     `json:"sorting"` // 1 - by reviews, 2 - price desc, 3 - price asc
	UserID         int     `json:"user_id,omitempty"`
}

type FilteredService struct {
	UserID             int     `json:"user_id"`
	UserName           string  `json:"user_name"`
	UserSurname        string  `json:"user_surname"`
	UserPhone          string  `json:"user_phone"`
	UserAvatarPath     *string `json:"user_avatar_path,omitempty"`
	UserRating         float64 `json:"user_rating"`
	UserReviewsCount   int     `json:"user_reviews_count"`
	ServiceID          int     `json:"service_id"`
	ServiceName        string  `json:"service_name"`
	ServicePrice       float64 `json:"service_price"`
	ServiceDescription string  `json:"service_description"`
	Liked              bool    `json:"liked"`
	Responded          bool    `json:"is_responded"`
}

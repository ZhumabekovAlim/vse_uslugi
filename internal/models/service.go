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
		ReviewRating float64 `json:"review_rating"`
	} `json:"user"`
	Images        string     `json:"images"`
	CategoryID    int        `json:"category_id, omitempty"`
	SubcategoryID int        `json:"subcategory_id, omitempty"`
	Description   string     `json:"description"`
	AvgRating     float64    `json:"avg_rating"`
	Top           string     `json:"top, omitempty"`
	Liked         bool       `json:"liked, omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type ServiceResponse struct {
	ClientID           int     `json:"client_id"`
	ClientName         string  `json:"client_name"`
	ClientRating       float64 `json:"client_rating"`
	ServiceID          int     `json:"service_id"`
	ServiceName        string  `json:"service_name"`
	ServicePrice       float64 `json:"service_price"`
	ServiceDescription string  `json:"service_description"`
}

type GetServicesRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	Sorting       string    `json:"sorting"`
	Page          int       `json:"page"`
	PageSize      int       `json:"page_size"`
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
}

type FilteredService struct {
	UserID             int     `json:"user_id"`
	UserName           string  `json:"user_name"`
	UserRating         float64 `json:"user_rating"`
	ServiceID          int     `json:"service_id"`
	ServiceName        string  `json:"service_name"`
	ServicePrice       float64 `json:"service_price"`
	ServiceDescription string  `json:"service_description"`
}

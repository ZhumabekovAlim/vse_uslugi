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
	Images      string     `json:"images"`
	CategoryID  int        `json:"category_id, omitempty"`
	Description string     `json:"description"`
	AvgRating   float64    `json:"avg_rating"`
	Liked       bool       `json:"liked, omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
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

type FilteredServiceRequest struct {
	Categories []struct {
		ID            int `json:"id"`
		Name          string
		Subcategories []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"subcategories"`
	} `json:"categories"`
	PriceFrom float64 `json:"price_from"`
	PriceTo   float64 `json:"price_to"`
	Ratings   []int   `json:"ratings"` // 1,2,3,4,5
	Sorting   int     `json:"sorting"` // 1,2,3
}

type FilteredServiceResponse struct {
	ClientID           int     `json:"client_id"`
	ClientName         string  `json:"client_name"`
	ClientRating       float64 `json:"client_rating"`
	ServiceID          int     `json:"service_id"`
	ServiceName        string  `json:"service_name"`
	ServicePrice       float64 `json:"service_price"`
	ServiceDescription string  `json:"service_description"`
}

type GetServicesPostRequest struct {
	Categories    []int    `json:"categories"`
	Subcategories []string `json:"subcategories"`
	PriceFrom     float64  `json:"price_from"`
	PriceTo       float64  `json:"price_to"`
	Ratings       []int    `json:"ratings"` // Пример: [3, 4, 5]
	Sorting       int      `json:"sorting"` // 1 - по отзывам, 2 - цена ↑, 3 - цена ↓
}

type GetServicesPostResponse struct {
	ClientID           int     `json:"client_id"`
	ClientName         string  `json:"client_name"`
	ClientReviewRating float64 `json:"client_rating"`
	ServiceID          int     `json:"service_id"`
	ServiceName        string  `json:"service_name"`
	ServicePrice       float64 `json:"service_price"`
	ServiceDescription string  `json:"service_description"`
}

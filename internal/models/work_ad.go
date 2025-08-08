package models

import (
	"time"
)

type WorkAd struct {
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
	Images          []ImageWorkAd `json:"images"`
	CategoryID      int           `json:"category_id, omitempty"`
	SubcategoryID   int           `json:"subcategory_id, omitempty"`
	Description     string        `json:"description"`
	AvgRating       float64       `json:"avg_rating"`
	Top             string        `json:"top, omitempty"`
	Liked           bool          `json:"liked, omitempty"`
	Status          string        `json:"status, omitempty"`
	CategoryName    string        `json:"category_name"`
	SubcategoryName string        `json:"subcategory_name"`
	WorkExperience  string        `json:"work_experience,omitempty"`
	CityID          int           `json:"city_id"`
	CityName        string        `json:"city_name"`
	CityType        string        `json:"city_type"`
	Schedule        string        `json:"schedule, omitempty"`
	DistanceWork    string        `json:"distance_work,omitempty"`
	PaymentPeriod   string        `json:"payment_period,omitempty"`
	Latitude        string        `json:"latitude,omitempty"`
	Longitude       string        `json:"longitude,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       *time.Time    `json:"updated_at,omitempty"`
}

type ImageWorkAd struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type WorkAdFilterRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	SortOption    int       `json:"sort"`
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
}

type WorkAdListResponse struct {
	WorksAd  []WorkAd `json:"works_ad"`
	MinPrice float64  `json:"min_price"`
	MaxPrice float64  `json:"max_price"`
}

type FilterWorkAdRequest struct {
	CategoryIDs    []int   `json:"category_id"`
	SubcategoryIDs []int   `json:"subcategory_id"`
	PriceFrom      float64 `json:"price_from"`
	PriceTo        float64 `json:"price_to"`
	AvgRatings     []int   `json:"avg_rating"`
	Sorting        int     `json:"sorting"` // 1 - by reviews, 2 - price desc, 3 - price asc
	UserID         int     `json:"user_id,omitempty"`
}

type FilteredWorkAd struct {
	UserID            int     `json:"user_id"`
	UserName          string  `json:"user_name"`
	UserRating        float64 `json:"user_rating"`
	WorkAdID          int     `json:"workad_id"`
	WorkAdName        string  `json:"workad_name"`
	WorkAdPrice       float64 `json:"workad_price"`
	WorkAdDescription string  `json:"workad_description"`
	Liked             bool    `json:"liked"`
}

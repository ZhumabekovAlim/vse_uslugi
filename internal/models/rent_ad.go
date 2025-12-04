package models

import (
	"time"
)

type RentAd struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	Price      *float64 `json:"price"`
	PriceTo    *float64 `json:"price_to"`
	Negotiable bool     `json:"negotiable"`
	HidePhone  bool     `json:"hide_phone"`
	UserID     int      `json:"user_id"`
	User       struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Surname      string  `json:"surname"`
		Phone        string  `json:"phone"`
		ReviewRating float64 `json:"review_rating"`
		ReviewsCount int     `json:"reviews_count"`
		AvatarPath   *string `json:"avatar_path,omitempty"`
	} `json:"user"`
	Images            []ImageRentAd `json:"images"`
	Videos            []Video       `json:"videos"`
	CategoryID        int           `json:"category_id, omitempty"`
	SubcategoryID     int           `json:"subcategory_id, omitempty"`
	Description       string        `json:"description"`
	WorkTimeFrom      string        `json:"work_time_from"`
	WorkTimeTo        string        `json:"work_time_to"`
	AvgRating         float64       `json:"avg_rating"`
	Top               string        `json:"top, omitempty"`
	Liked             bool          `json:"liked, omitempty"`
	Responded         bool          `json:"is_responded"`
	Status            string        `json:"status, omitempty"`
	CategoryName      string        `json:"category_name"`
	SubcategoryName   string        `json:"subcategory_name"`
	SubcategoryNameKz string        `json:"subcategory_name_kz"`
	RentType          string        `json:"rent_type"`
	Deposit           string        `json:"deposit"`
	Latitude          string        `json:"latitude,omitempty"`
	Longitude         string        `json:"longitude,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         *time.Time    `json:"updated_at,omitempty"`
}

type ImageRentAd struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type RentAdFilterRequest struct {
	Categories    []int     `json:"categories"`
	Subcategories []string  `json:"subcategories"`
	PriceFrom     float64   `json:"price_from"`
	PriceTo       float64   `json:"price_to"`
	Ratings       []float64 `json:"ratings"`
	SortOption    int       `json:"sort"`
	Page          int       `json:"page"`
	Limit         int       `json:"limit"`
}

type RentAdListResponse struct {
	RentsAd  []RentAd `json:"rents_ad"`
	MinPrice float64  `json:"min_price"`
	MaxPrice float64  `json:"max_price"`
}

type FilterRentAdRequest struct {
	CategoryIDs     []int    `json:"category_id"`
	SubcategoryIDs  []int    `json:"subcategory_id"`
	PriceFrom       float64  `json:"price_from"`
	PriceTo         float64  `json:"price_to"`
	AvgRatings      []int    `json:"avg_rating"`
	Negotiable      *bool    `json:"negotiable,omitempty"`
	RentTypes       []string `json:"rent_type"`
	Deposits        []string `json:"deposit"`
	OpenNow         bool     `json:"open_now"`
	TwentyFourSeven bool     `json:"twenty_four_seven"`
	Sorting         int      `json:"sorting"` // 1 - by reviews, 2 - price desc, 3 - price asc
	UserID          int      `json:"user_id,omitempty"`
	CityID          int      `json:"city_id,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	RadiusKm        *float64 `json:"radius_km,omitempty"`
}

type FilteredRentAd struct {
	UserID            int           `json:"user_id"`
	UserName          string        `json:"user_name"`
	UserSurname       string        `json:"user_surname"`
	UserPhone         string        `json:"-"`
	UserAvatarPath    *string       `json:"user_avatar_path,omitempty"`
	UserRating        float64       `json:"user_rating"`
	UserReviewsCount  int           `json:"user_reviews_count"`
	RentAdID          int           `json:"rentad_id"`
	RentAdName        string        `json:"rentad_name"`
	RentAdAddress     string        `json:"rentad_address"`
	RentAdPrice       *float64      `json:"rentad_price"`
	RentAdPriceTo     *float64      `json:"price_to"`
	RentAdNegotiable  bool          `json:"negotiable"`
	RentAdHidePhone   bool          `json:"hide_phone"`
	RentAdDescription string        `json:"rentad_description"`
	WorkTimeFrom      string        `json:"work_time_from"`
	WorkTimeTo        string        `json:"work_time_to"`
	Images            []ImageRentAd `json:"images"`
	Videos            []Video       `json:"videos"`
	RentAdLatitude    string        `json:"latitude"`
	RentAdLongitude   string        `json:"longitude"`
	Distance          *float64      `json:"distance,omitempty"`
	Liked             bool          `json:"liked"`
	Responded         bool          `json:"is_responded"`
	Top               string        `json:"-"`
	CreatedAt         time.Time     `json:"-"`
}

package models

// Location represents a user's geographic coordinates.
type Location struct {
	UserID    int      `json:"user_id"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ExecutorLocationFilter defines filters for fetching executors on map.
type ExecutorLocationFilter struct {
	CategoryIDs    []int     `json:"category_id"`
	SubcategoryIDs []int     `json:"subcategory_id"`
	PriceFrom      float64   `json:"price_from"`
	PriceTo        float64   `json:"price_to"`
	AvgRating      []float64 `json:"avg_rating"`
	Type           string    `json:"-"`
}

// ExecutorLocationItem describes a single active item bound to an executor.
type ExecutorLocationItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	AvgRating   float64 `json:"avg_rating"`
}

// ExecutorLocationGroup aggregates an executor's profile with all active items.
type ExecutorLocationGroup struct {
	UserID    int      `json:"user_id"`
	Name      string   `json:"name"`
	Surname   string   `json:"surname"`
	Avatar    *string  `json:"avatar,omitempty"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	Services []ExecutorLocationItem `json:"services"`
	Ads      []ExecutorLocationItem `json:"ads"`
	WorkAds  []ExecutorLocationItem `json:"work_ads"`
	Works    []ExecutorLocationItem `json:"works"`
	RentAds  []ExecutorLocationItem `json:"rent_ads"`
	Rents    []ExecutorLocationItem `json:"rents"`
}

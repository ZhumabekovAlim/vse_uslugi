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
}

// ExecutorLocation represents executor with current item and coordinates.
type ExecutorLocation struct {
	UserID      int      `json:"user_id"`
	Name        string   `json:"name"`
	Surname     string   `json:"surname"`
	Avatar      *string  `json:"avatar,omitempty"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	ItemID      int      `json:"item_id"`
	ItemName    string   `json:"item_name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	AvgRating   float64  `json:"avg_rating"`
	Type        string   `json:"type"`
}

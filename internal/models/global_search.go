package models

// GlobalSearchRequest describes filters for the global search endpoint.
type GlobalSearchRequest struct {
	Types          []string  `json:"types"`
	CategoryIDs    []int     `json:"category_ids"`
	SubcategoryIDs []int     `json:"subcategory_ids"`
	Limit          int       `json:"limit"`
	Page           int       `json:"page"`
	PriceFrom      float64   `json:"price_from"`
	PriceTo        float64   `json:"price_to"`
	Ratings        []float64 `json:"ratings"`
	SortOption     int       `json:"sort_option"`
	OnSite         *bool     `json:"on_site,omitempty"`
	Negotiable     *bool     `json:"negotiable,omitempty"`
	RentTypes      []string  `json:"rent_types,omitempty"`
	Deposits       []string  `json:"deposits,omitempty"`
	WorkExperience []string  `json:"work_experience,omitempty"`
	WorkSchedules  []string  `json:"work_schedules,omitempty"`
	PaymentPeriods []string  `json:"payment_periods,omitempty"`
	RemoteWork     *bool     `json:"remote_work,omitempty"`
	Languages      []string  `json:"languages,omitempty"`
	Educations     []string  `json:"educations,omitempty"`
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	RadiusKm       *float64  `json:"radius_km,omitempty"`
	UserID         int       `json:"-"`
}

// GlobalSearchItem represents a single listing returned by the global search.
type GlobalSearchItem struct {
	Type     string   `json:"type"`
	Distance *float64 `json:"distance,omitempty"`
	Service  *Service `json:"service,omitempty"`
	Ad       *Ad      `json:"ad,omitempty"`
	Work     *Work    `json:"work,omitempty"`
	WorkAd   *WorkAd  `json:"work_ad,omitempty"`
	Rent     *Rent    `json:"rent,omitempty"`
	RentAd   *RentAd  `json:"rent_ad,omitempty"`
}

// GlobalSearchResponse is a paginated response for the global search endpoint.
type GlobalSearchResponse struct {
	Results []GlobalSearchItem `json:"results"`
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	Limit   int                `json:"limit"`
}

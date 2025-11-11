package models

// GlobalSearchRequest describes filters for the global search endpoint.
type GlobalSearchRequest struct {
	Types          []string `json:"types"`
	CategoryIDs    []int    `json:"category_ids"`
	SubcategoryIDs []int    `json:"subcategory_ids"`
	Limit          int      `json:"limit"`
	Page           int      `json:"page"`
	UserID         int      `json:"-"`
}

// GlobalSearchItem represents a single listing returned by the global search.
type GlobalSearchItem struct {
	Type    string   `json:"type"`
	Service *Service `json:"service,omitempty"`
	Ad      *Ad      `json:"ad,omitempty"`
	Work    *Work    `json:"work,omitempty"`
	WorkAd  *WorkAd  `json:"work_ad,omitempty"`
	Rent    *Rent    `json:"rent,omitempty"`
	RentAd  *RentAd  `json:"rent_ad,omitempty"`
}

// GlobalSearchResponse is a paginated response for the global search endpoint.
type GlobalSearchResponse struct {
	Results []GlobalSearchItem `json:"results"`
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	Limit   int                `json:"limit"`
}

package models

import "time"

type AdsFilter struct {
	Type          string  `json:"type"`
	CategoryID    int     `json:"category_id"`
	SubcategoryID int     `json:"subcategory_id"`
	MinPrice      float64 `json:"min_price"`
	MaxPrice      float64 `json:"max_price"`
	Search        string  `json:"search"`
	Page          int     `json:"page"`
	PageSize      int     `json:"page_size"`
}

type AdsList struct {
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Total    int      `json:"total"`
	Items    []AdItem `json:"items"`
}

type AdItem struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Address     string    `json:"address"`
	CreatedAt   time.Time `json:"created_at"`
	Category    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"category"`
	Subcategory struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"subcategory"`
	ViewsCount     int `json:"views_count"`
	ResponsesCount int `json:"responses_count"`
	Author         struct {
		ID       int     `json:"id"`
		Name     string  `json:"name"`
		Rating   float64 `json:"rating"`
		Phone    string  `json:"phone"`
		ChatLink string  `json:"chat_link"`
	} `json:"author"`
	WorkScope        *string             `json:"work_scope,omitempty"`
	DepositRequired  *string             `json:"deposit_required,omitempty"`
	RentalTerms      *string             `json:"rental_terms,omitempty"`
	EmploymentType   *string             `json:"employment_type,omitempty"`
	SalaryFrom       *float64            `json:"salary_from,omitempty"`
	SalaryTo         *float64            `json:"salary_to,omitempty"`
	ResponsesPreview []AdResponsePreview `json:"responses_preview"`
}

type AdResponsePreview struct {
	ID   int `json:"id"`
	User struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Rating float64 `json:"rating"`
	} `json:"user"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

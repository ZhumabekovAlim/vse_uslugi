package models

import "time"

// ChatUser contains information about a user participating in a chat and the price they offered.
type ChatUser struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Surname       string          `json:"surname"`
	AvatarPath    string          `json:"avatar_path"`
	Phone         string          `json:"phone,omitempty"`
	ProviderPhone string          `json:"provider_phone"`
	ClientPhone   string          `json:"client_phone"`
	Price         float64         `json:"price"`
	ChatID        int             `json:"chat_id"`
	LastMessage   string          `json:"lastMessage,omitempty"`
	LastMessageAt *time.Time      `json:"lastMessageAt,omitempty"`
	ReviewRating  float64         `json:"review_rating"`
	ReviewsCount  int             `json:"reviews_count"`
	MyRole        string          `json:"my_role"`
	AdReview      *ChatUserReview `json:"ad_review,omitempty"`
}

// ChatUserReview describes a review left by a user for a particular advertisement.
type ChatUserReview struct {
	UserID int     `json:"user_id"`
	Rating float64 `json:"rating"`
	Review string  `json:"review"`
}

// AdChats groups chat users by advertisement.
type AdChats struct {
	AdType         string     `json:"ad_type,omitempty"`
	AdID           *int       `json:"ad_id,omitempty"`
	ServiceID      *int       `json:"service_id,omitempty"`
	RentAdID       *int       `json:"rentad_id,omitempty"`
	WorkAdID       *int       `json:"workad_id,omitempty"`
	RentID         *int       `json:"rent_id,omitempty"`
	WorkID         *int       `json:"work_id,omitempty"`
	AdName         string     `json:"ad_name"`
	Status         string     `json:"status"`
	IsAuthor       bool       `json:"is_author"`
	HidePhone      bool       `json:"hide_phone"`
	PerformerID    *int       `json:"performer_id,omitempty"`
	Address        string     `json:"address,omitempty"`
	Price          *float64   `json:"price,omitempty"`
	PriceTo        *float64   `json:"price_to,omitempty"`
	Negotiable     bool       `json:"negotiable"`
	OnSite         bool       `json:"on_site,omitempty"`
	Description    string     `json:"description,omitempty"`
	WorkTimeFrom   string     `json:"work_time_from,omitempty"`
	WorkTimeTo     string     `json:"work_time_to,omitempty"`
	Latitude       *string    `json:"latitude,omitempty"`
	Longitude      *string    `json:"longitude,omitempty"`
	RentType       string     `json:"rent_type,omitempty"`
	Deposit        string     `json:"deposit,omitempty"`
	WorkExperience string     `json:"work_experience,omitempty"`
	Schedule       string     `json:"schedule,omitempty"`
	DistanceWork   string     `json:"distance_work,omitempty"`
	PaymentPeriod  string     `json:"payment_period,omitempty"`
	Images         []Image    `json:"images,omitempty"`
	Videos         []Video    `json:"videos,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	Users          []ChatUser `json:"users"`
}

// BusinessWorkerChat represents chats between a business and its workers without advertisement metadata.
type BusinessWorkerChat struct {
	ChatID       int        `json:"chat_id"`
	WorkerUserID int        `json:"worker_user_id"`
	Login        string     `json:"login"`
	Users        []ChatUser `json:"users"`
}

// SetIDByType assigns the advertisement identifier based on its type.
func (a *AdChats) SetIDByType(adType string, id int) {
	switch adType {
	case "service":
		a.ServiceID = intPtr(id)
	case "rent_ad":
		a.RentAdID = intPtr(id)
	case "work_ad":
		a.WorkAdID = intPtr(id)
	case "rent":
		a.RentID = intPtr(id)
	case "work":
		a.WorkID = intPtr(id)
	default:
		a.AdID = intPtr(id)
	}
}

func intPtr(value int) *int {
	v := value
	return &v
}

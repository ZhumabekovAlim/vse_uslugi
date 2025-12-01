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
	AdID        *int       `json:"ad_id,omitempty"`
	ServiceID   *int       `json:"service_id,omitempty"`
	RentAdID    *int       `json:"rentad_id,omitempty"`
	WorkAdID    *int       `json:"workad_id,omitempty"`
	RentID      *int       `json:"rent_id,omitempty"`
	WorkID      *int       `json:"work_id,omitempty"`
	AdName      string     `json:"ad_name"`
	Status      string     `json:"status"`
	IsAuthor    bool       `json:"is_author"`
	HidePhone   bool       `json:"hide_phone"`
	PerformerID *int       `json:"performer_id,omitempty"`
	Users       []ChatUser `json:"users"`
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

package models

// ChatUser contains information about a user participating in a chat and the price they offered.
type ChatUser struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Surname string  `json:"surname"`
	Price   float64 `json:"price"`
	ChatID  int     `json:"chat_id"`
}

// AdChats groups chat users by advertisement.
type AdChats struct {
	AdID   int        `json:"ad_id"`
	AdName string     `json:"ad_name"`
	Users  []ChatUser `json:"users"`
}

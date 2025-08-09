package models

import "time"

// Структура чата
type Chat struct {
	ID      int `json:"id"`
	User1ID int `json:"user1_id"`
	User2ID int `json:"user2_id"`
	User1   struct {
		Name       string  `json:"name"`
		Surname    string  `json:"surname"`
		AvatarPath *string `json:"avatar_path,omitempty"`
	} `json:"user1"`
	User2 struct {
		Name       string  `json:"name"`
		Surname    string  `json:"surname"`
		AvatarPath *string `json:"avatar_path,omitempty"`
	} `json:"user2"`
	CreatedAt time.Time `json:"created_at"`
}

package models

import "time"

// Структура сообщения
type Message struct {
	ID         string    `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	ChatID     int       `json:"chat_id"`
}

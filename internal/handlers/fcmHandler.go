package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"firebase.google.com/go/messaging"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"log"
	_ "naimuBack/internal/models"
	"net/http"
)

type FCMHandler struct {
	Client *messaging.Client
	DB     *sql.DB
}

type NotificationRequest struct {
	Id       int    `json:"id"`
	UserId   int    `json:"user_id"`
	Token    string `json:"token"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Sender   int    `json:"sender"`
	Receiver int    `json:"receiver"`
	Link     string `json:"link"`
	Param1   string `json:"param1"`
	Param2   string `json:"param2"`
}

type Token struct {
	UserId int    `json:"user_id"`
	Token  string `json:"token"`
}

func NewFCMHandler(client *messaging.Client, db *sql.DB) *FCMHandler {
	return &FCMHandler{Client: client, DB: db}
}

func (h *FCMHandler) SendMessage(ctx context.Context, token string, UserId, sender, receiver int, title, body, link, param1, param2 string) error {
	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"link":   link,
			"param1": param1,
			"param2": param2,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "high_priority_channel",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Sound: "default",
				},
			},
		},
	}

	response, err := h.Client.Send(ctx, message)
	if err != nil {
		log.Printf("Ошибка при отправке уведомления: %v", err)
		return err
	}

	log.Printf("Отправка уведомления выполнена успешно: %s\n", response)
	return nil
}

func (h *FCMHandler) NotifyChange(w http.ResponseWriter, r *http.Request) {
	var req NotificationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received notification request: %+v", req)

	ctx := r.Context()
	tokens, err := h.GetTokensByClientID(req.UserId)
	if err != nil {
		log.Printf("Error fetching tokens: %v", err)
		http.Error(w, "Failed to fetch tokens", http.StatusInternalServerError)
		return
	}

	// Send notifications to each token
	for _, token := range tokens {
		err = h.SendMessage(ctx, token, req.UserId, req.Sender, req.Receiver, req.Title, req.Body, req.Link, req.Param1, req.Param2)
		if err != nil {
			log.Printf("Error sending notification to token %s: %v", token, err)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Notification sent successfully"))
	if err != nil {
		return
	}
}

func (h *FCMHandler) GetTokensByClientID(clientID int) ([]string, error) {
	if h.DB == nil {
		log.Print("h.DB is nil")
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var tokens []string

	baseQuery := "SELECT token FROM notify_tokens WHERE user_id = ?"
	var args []interface{}
	args = append(args, clientID)

	rows, err := h.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (h *FCMHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var newToken Token

	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	err := json.NewDecoder(r.Body).Decode(&newToken)
	if err != nil {
		http.Error(w, "Failed to fetch tokens", http.StatusBadRequest)
		return
	}

	err = h.InsertToken(newToken.UserId, newToken.Token)
	if err != nil {
		http.Error(w, "Failed to insert tokens", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *FCMHandler) InsertToken(clientID int, token string) error {

	stmt1 := `
        INSERT INTO notify_tokens 
        (user_id, token) 
        VALUES ( ?, ?);`

	_, err := h.DB.Exec(stmt1, clientID, token)
	if err != nil {
		return err
	}
	return nil
}

func (h *FCMHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get(":token")

	if token == "" {
		http.Error(w, "Failed to fetch tokens", http.StatusInternalServerError)
		return
	}

	err := h.DeleteTokenRep(token)
	if err != nil {
		http.Error(w, "Failed to delete tokens", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *FCMHandler) DeleteTokenRep(token string) error {
	stmt := `DELETE FROM notify_tokens WHERE token = ?`
	_, err := h.DB.Exec(stmt, token)
	if err != nil {
		return err
	}

	return nil
}

func (h *FCMHandler) NotifyChangeForAll(w http.ResponseWriter, r *http.Request) {
	var req NotificationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tokens, err := h.GetAllTokens()
	if err != nil {
		http.Error(w, "Failed to fetch tokens", http.StatusInternalServerError)
		return
	}

	if len(tokens) == 0 {
		http.Error(w, "No tokens found", http.StatusNotFound)
		return
	}

	// Send notification to each token
	for _, token := range tokens {
		err := h.SendMessage(ctx, token, req.UserId, req.Sender, req.Receiver, req.Title, req.Body, req.Link, req.Param1, req.Param2)
		if err != nil {
			log.Printf("Error sending notification to token %s: %v", token, err)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Notification sent successfully"))
}

func (h *FCMHandler) GetAllTokens() ([]string, error) {
	if h.DB == nil {
		log.Print("h.DB is nil")
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var tokens []string
	query := "SELECT token FROM notify_tokens"
	rows, err := h.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

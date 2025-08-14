package main

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql" // MariaDB/MySQL
	"github.com/gorilla/websocket"
	"log"
	"naimuBack/internal/models"
	"net/http"
	"strings"
	"time"
)

// --- WS manager ---

type directMsg struct {
	userID int
	msg    models.Message
}

type WebSocketManager struct {
	clients    map[int]*websocket.Conn
	broadcast  chan models.Message
	direct     chan directMsg
	register   chan Client
	unregister chan int
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[int]*websocket.Conn),
		broadcast:  make(chan models.Message),
		direct:     make(chan directMsg),
		register:   make(chan Client),
		unregister: make(chan int),
	}
}

type Client struct {
	ID     int
	Socket *websocket.Conn
}

// Все операции с clients — только здесь, чтобы не было гонок.
func (ws *WebSocketManager) Run(_ *sql.DB) {
	for {
		select {
		case client := <-ws.register:
			ws.clients[client.ID] = client.Socket

		case clientID := <-ws.unregister:
			if conn, ok := ws.clients[clientID]; ok {
				_ = conn.Close()
				delete(ws.clients, clientID)
			}

		case msg := <-ws.broadcast:
			for id, conn := range ws.clients {
				if err := conn.WriteJSON(msg); err != nil {
					log.Println("broadcast error:", err)
					_ = conn.Close()
					delete(ws.clients, id)
				}
			}

		case dm := <-ws.direct:
			if conn, ok := ws.clients[dm.userID]; ok {
				if err := conn.WriteJSON(dm.msg); err != nil {
					log.Println("direct send error:", err)
					_ = conn.Close()
					delete(ws.clients, dm.userID)
				}
			}
		}
	}
}

// --- WS handler ---

var upgrader = websocket.Upgrader{
	// Если нужен жёсткий контроль — проверь Origin здесь.
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Первым фреймом клиент обязан прислать { "userId": <int> }.
func (app *application) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// read deadlines + pong handler
	conn.SetReadLimit(1 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var hello struct {
		UserID int `json:"userId"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.UserID == 0 {
		log.Println("invalid hello payload:", err)
		_ = conn.Close()
		return
	}

	client := Client{ID: hello.UserID, Socket: conn}
	app.wsManager.register <- client

	// периодический ping, чтобы держать коннект живым
	go func(c *websocket.Conn, uid int) {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				// закроем и дадим менеджеру убрать клиента
				app.wsManager.unregister <- uid
				return
			}
		}
	}(conn, hello.UserID)

	// читаем сообщения пользователя
	go handleWebSocketMessages(conn, hello.UserID, app.wsManager, app.db)
}

func handleWebSocketMessages(conn *websocket.Conn, userID int, wsManager *WebSocketManager, db *sql.DB) {
	defer func() {
		wsManager.unregister <- userID
		_ = conn.Close()
	}()

	for {
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read json error:", err)
			return
		}

		// простая валидация
		if msg.SenderID != userID {
			log.Println("reject: senderID != authenticated userID")
			continue
		}
		if msg.ReceiverID == 0 || strings.TrimSpace(msg.Text) == "" {
			log.Println("reject: empty receiver or text")
			continue
		}

		msg.CreatedAt = time.Now()

		// получаем/создаем чат и сохраняем сообщение
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		chatID, err := getOrCreateChat(ctx, db, msg.SenderID, msg.ReceiverID)
		cancel()
		if err != nil {
			log.Println("get/create chat error:", err)
			continue
		}
		msg.ChatID = chatID

		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		if err := saveMessageToDB(ctx, db, msg); err != nil {
			cancel()
			log.Println("save message error:", err)
			continue
		}
		cancel()

		// отправка получателю (через менеджер, без прямого доступа к карте)
		wsManager.direct <- directMsg{userID: msg.ReceiverID, msg: msg}
	}
}

// --- DB helpers (MariaDB / MySQL) ---

func getOrCreateChat(ctx context.Context, db *sql.DB, user1ID, user2ID int) (int, error) {
	// 1) пробуем найти
	var chatID int
	err := db.QueryRowContext(ctx, `
		SELECT id FROM chats
		WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)
		LIMIT 1
	`, user1ID, user2ID, user2ID, user1ID).Scan(&chatID)
	if err == nil {
		return chatID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	// 2) создаём
	res, err := db.ExecContext(ctx, `
		INSERT INTO chats (user1_id, user2_id) VALUES (?, ?)
	`, user1ID, user2ID)
	if err != nil {
		// возможен race при одновременном создании — пробуем перечитать
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return getOrCreateChat(ctx, db, user1ID, user2ID)
		}
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

func saveMessageToDB(ctx context.Context, db *sql.DB, msg models.Message) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO messages (chat_id, sender_id, receiver_id, text, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, msg.ChatID, msg.SenderID, msg.ReceiverID, msg.Text, msg.CreatedAt)
	return err
}

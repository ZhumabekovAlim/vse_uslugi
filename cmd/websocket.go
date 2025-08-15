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

/********** настройки таймингов **********/
const (
	readLimit          = 1 << 20           // 1MB
	readDeadline       = 120 * time.Second // сколько ждём следующего сообщения/ponг’а
	writeDeadline      = 5 * time.Second   // дедлайн на запись кадра
	pingInterval       = 15 * time.Second  // как часто пингуем
	firstHelloDeadline = 30 * time.Second  // время на первый кадр {userId}
)

/*****************************************/

type directMsg struct {
	userID int
	msg    models.Message
}

type unreg struct {
	userID int
	conn   *websocket.Conn
}

type WebSocketManager struct {
	clients    map[int]*websocket.Conn
	broadcast  chan models.Message
	direct     chan directMsg
	register   chan Client
	unregister chan unreg
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[int]*websocket.Conn),
		broadcast:  make(chan models.Message),
		direct:     make(chan directMsg),
		register:   make(chan Client),
		unregister: make(chan unreg),
	}
}

type Client struct {
	ID     int
	Socket *websocket.Conn
}

// Все операции с clients — только здесь.
func (ws *WebSocketManager) Run(_ *sql.DB) {
	for {
		select {
		case client := <-ws.register:
			// если был старый сокет — гасим (чтобы его тикер/ридер не мешали)
			if old, ok := ws.clients[client.ID]; ok && old != nil && old != client.Socket {
				_ = old.Close()
			}
			ws.clients[client.ID] = client.Socket
			log.Printf("WS register user=%d", client.ID)

		case u := <-ws.unregister:
			// удаляем только если совпадает текущий сокет пользователя
			if cur, ok := ws.clients[u.userID]; ok && cur == u.conn {
				_ = cur.Close()
				delete(ws.clients, u.userID)
				log.Printf("WS unregister user=%d", u.userID)
			}

		case msg := <-ws.broadcast:
			for id, conn := range ws.clients {
				_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("broadcast error to=%d: %v", id, err)
					_ = conn.Close()
					delete(ws.clients, id)
				}
			}

		case dm := <-ws.direct:
			if conn, ok := ws.clients[dm.userID]; ok {
				_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
				if err := conn.WriteJSON(dm.msg); err != nil {
					log.Printf("direct send error to=%d: %v", dm.userID, err)
					_ = conn.Close()
					delete(ws.clients, dm.userID)
				}
			} else {
				// получатель оффлайн — это нормально
				log.Printf("direct skip: user=%d offline", dm.userID)
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:       func(r *http.Request) bool { return true }, // при желании — белый список Origin
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: true,
}

// Первым фреймом клиент обязан прислать { "userId": <int> }.
func (app *application) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// ограничение и дедлайны чтения
	conn.SetReadLimit(readLimit)
	conn.SetReadDeadline(time.Now().Add(firstHelloDeadline)) // время на hello
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// hello
	var hello struct {
		UserID int `json:"userId"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.UserID == 0 {
		log.Println("invalid hello payload:", err)
		_ = writeClose(conn, websocket.ClosePolicyViolation, "hello required")
		_ = conn.Close()
		return
	}
	// после hello продлеваем чтение
	conn.SetReadDeadline(time.Now().Add(readDeadline))

	client := Client{ID: hello.UserID, Socket: conn}
	app.wsManager.register <- client

	// ping-тример (живём пока запись удаётся)
	go pingLoop(app.wsManager, conn, hello.UserID)

	// читаем сообщения
	go handleWebSocketMessages(conn, hello.UserID, app.wsManager, app.db)
}

func pingLoop(ws *WebSocketManager, conn *websocket.Conn, uid int) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for range t.C {
		_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			// аккуратно закрываем и снимаем только если это актуальный сокет
			_ = writeClose(conn, websocket.CloseGoingAway, "ping error")
			ws.unregister <- unreg{userID: uid, conn: conn}
			return
		}
	}
}

func handleWebSocketMessages(conn *websocket.Conn, userID int, wsManager *WebSocketManager, db *sql.DB) {
	defer func() {
		wsManager.unregister <- unreg{userID: userID, conn: conn}
		_ = conn.Close()
	}()

	for {
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read json error:", err)
			_ = writeClose(conn, websocket.CloseNormalClosure, "read error")
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

		// получаем/создаём чат
		{
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			chatID, err := getOrCreateChat(ctx, db, msg.SenderID, msg.ReceiverID)
			cancel()
			if err != nil {
				log.Println("get/create chat error:", err)
				continue
			}
			msg.ChatID = chatID
		}

		// сохраняем
		{
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := saveMessageToDB(ctx, db, msg); err != nil {
				cancel()
				log.Println("save message error:", err)
				continue
			}
			cancel()
		}

		// доставляем получателю
		wsManager.direct <- directMsg{userID: msg.ReceiverID, msg: msg}
	}
}

// аккуратная отправка close-фрейма
func writeClose(conn *websocket.Conn, code int, reason string) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	return conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(writeDeadline),
	)
}

/********** DB helpers (MariaDB / MySQL) **********/

func getOrCreateChat(ctx context.Context, db *sql.DB, user1ID, user2ID int) (int, error) {
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

	res, err := db.ExecContext(ctx, `INSERT INTO chats (user1_id, user2_id) VALUES (?, ?)`, user1ID, user2ID)
	if err != nil {
		// гонка при одновременном создании
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

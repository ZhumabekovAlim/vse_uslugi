package main

import (
	"database/sql"
	"flag"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"log"
	"naimuBack/internal/config"
	"net/http"
	"os"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	cfg := config.LoadConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = ":4001"
	} else {
		port = ":" + port
	}

	addr := flag.String("addr", port, "HTTP network address")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(cfg.Database.URL)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	app := initializeApp(db, errorLog, infoLog)

	app.wsManager = NewWebSocketManager()
	go app.wsManager.Run(db)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      addSecurityHeaders(c.Handler(app.routes())),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		errorLog.Fatal(err)
	}
}

func (ws *WebSocketManager) Run(db *sql.DB) {
	for {
		select {
		case client := <-ws.register:
			ws.clients[client.ID] = client.Socket
		case clientID := <-ws.unregister:
			if conn, ok := ws.clients[clientID]; ok {
				conn.Close()
				delete(ws.clients, clientID)
			}
		case msg := <-ws.broadcast:
			// Отправка сообщения всем клиентам
			for id, conn := range ws.clients {
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Error sending message:", err)
					ws.unregister <- id
				}
			}
		}
	}
}

//func (ws *WebSocketManager) Run(db *sql.DB) {
//	for {
//		select {
//		case client := <-ws.register:
//			ws.clients[client.ID] = client.Socket
//		case clientID := <-ws.unregister:
//			if conn, ok := ws.clients[clientID]; ok {
//				conn.Close()
//				delete(ws.clients, clientID)
//			}
//		case msg := <-ws.broadcast:
//			// Отправка сообщения всем клиентам
//			for id, conn := range ws.clients {
//				err := conn.WriteJSON(msg)
//				if err != nil {
//					log.Println("Error sending message:", err)
//					ws.unregister <- id
//				}
//			}
//		}
//	}
//}
//
//
//type WebSocketManager struct {
//	clients    map[int]*websocket.Conn
//	broadcast  chan models.Message
//	register   chan Client
//	unregister chan int
//}
//
//type Client struct {
//	ID     int
//	Socket *websocket.Conn
//}
//
//func NewWebSocketManager() *WebSocketManager {
//	return &WebSocketManager{
//		clients:    make(map[int]*websocket.Conn),
//		broadcast:  make(chan models.Message),
//		register:   make(chan Client),
//		unregister: make(chan int),
//	}
//}
//
//// WebSocket Handler для установки WebSocket соединений
//func (app *application) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
//	upgrader := websocket.Upgrader{
//		CheckOrigin: func(r *http.Request) bool {
//			return true
//		},
//	}
//
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		log.Println("WebSocket upgrade error:", err)
//		return
//	}
//
//	var clientData struct {
//		UserID int `json:"userId"`
//	}
//	err = conn.ReadJSON(&clientData)
//	if err != nil {
//		log.Println("Failed to read client data:", err)
//		conn.Close()
//		return
//	}
//
//	client := Client{
//		ID:     clientData.UserID,
//		Socket: conn,
//	}
//	app.wsManager.register <- client
//
//	// Обработка сообщений от клиента
//	go handleWebSocketMessages(conn, clientData.UserID, app.wsManager, app.db)
//}
//
//func handleWebSocketMessages(conn *websocket.Conn, userID int, wsManager *WebSocketManager, db *sql.DB) {
//	defer func() {
//		wsManager.unregister <- userID
//		conn.Close()
//	}()
//
//	for {
//		var msg models.Message
//		err := conn.ReadJSON(&msg)
//		if err != nil {
//			log.Println("Error reading message:", err)
//			break
//		}
//
//		// Генерация уникального ID для сообщения
//		msg.CreatedAt = time.Now()
//
//		// Получаем или создаем чат
//		chatID, err := getChatID(db, msg.SenderID, msg.ReceiverID)
//		if err != nil {
//			log.Println("Error getting chat ID:", err)
//			break
//		}
//
//		if chatID == 0 {
//			chatID, err = createChat(db, msg.SenderID, msg.ReceiverID)
//			if err != nil {
//				log.Println("Error creating chat:", err)
//				break
//			}
//		}
//
//		// Сохраняем сообщение в базе данных
//		msg.ChatID = chatID
//		saveMessageToDB(db, msg)
//
//		// Отправляем сообщение через WebSocket
//		if conn, ok := wsManager.clients[msg.ReceiverID]; ok {
//			err := conn.WriteJSON(msg)
//			if err != nil {
//				log.Println("Error sending message:", err)
//				wsManager.unregister <- msg.ReceiverID
//			}
//		}
//	}
//}
//
//// Генерация уникального ID для сообщения
//func generateMessageID() string {
//	return time.Now().Format("200601")
//}
//
//func randomString(length int) string {
//	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
//	var sb strings.Builder
//	for i := 0; i < length; i++ {
//		randomIndex := rand.Intn(len(charset))
//		sb.WriteByte(charset[randomIndex])
//	}
//	return sb.String()
//}
//
//func getChatID(db *sql.DB, user1ID, user2ID int) (int, error) {
//	var chatID int
//	err := db.QueryRow(`
//		SELECT id FROM chats
//		WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)`,
//		user1ID, user2ID, user2ID, user1ID).Scan(&chatID)
//	if err != nil && err != sql.ErrNoRows {
//		return 0, err
//	}
//	return chatID, nil
//}
//
//func createChat(db *sql.DB, user1ID, user2ID int) (int, error) {
//	result, err := db.Exec(`
//		INSERT INTO chats (user1_id, user2_id)
//		VALUES (?, ?)`, user1ID, user2ID)
//	if err != nil {
//		return 0, err
//	}
//	chatID, err := result.LastInsertId()
//	if err != nil {
//		return 0, err
//	}
//	return int(chatID), nil
//}
//
//func saveMessageToDB(db *sql.DB, msg models.Message) {
//	fmt.Println("Saving message to DB:", msg)
//	_, err := db.Exec(`
//		INSERT INTO messages (sender_id, receiver_id, text, created_at, chat_id)
//		VALUES ( ?, ?, ?, ?, ?)`,
//		msg.SenderID, msg.ReceiverID, msg.Text, msg.CreatedAt, msg.ChatID)
//	if err != nil {
//		log.Println("Error saving message to database 111:", err)
//	}
//}

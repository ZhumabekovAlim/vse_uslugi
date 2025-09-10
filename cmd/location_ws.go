package main

import (
	"context"
	"log"
	"naimuBack/internal/models"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// LocationManager manages websocket connections for location sharing.
type LocationManager struct {
	clients    map[int]*websocket.Conn
	register   chan Client
	unregister chan unreg
	broadcast  chan models.Location
}

// NewLocationManager creates a new LocationManager instance.
func NewLocationManager() *LocationManager {
	return &LocationManager{
		clients:    make(map[int]*websocket.Conn),
		register:   make(chan Client),
		unregister: make(chan unreg),
		broadcast:  make(chan models.Location),
	}
}

// Run starts the manager loop.
func (lm *LocationManager) Run() {
	for {
		select {
		case client := <-lm.register:
			if old, ok := lm.clients[client.ID]; ok && old != nil && old != client.Socket {
				_ = old.Close()
			}
			lm.clients[client.ID] = client.Socket
		case u := <-lm.unregister:
			if cur, ok := lm.clients[u.userID]; ok && cur == u.conn {
				_ = cur.Close()
				delete(lm.clients, u.userID)
			}
		case loc := <-lm.broadcast:
			for id, conn := range lm.clients {
				_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
				if err := conn.WriteJSON(loc); err != nil {
					_ = conn.Close()
					delete(lm.clients, id)
				}
			}
		}
	}
}

// LocationWebSocketHandler handles websocket connections for location updates.
func (app *application) LocationWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Location WS upgrade error:", err)
		return
	}

	conn.SetReadLimit(readLimit)
	conn.SetReadDeadline(time.Now().Add(firstHelloDeadline))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	var hello struct {
		UserID int `json:"userId"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.UserID == 0 {
		log.Println("invalid hello payload for location:", err)
		_ = writeClose(conn, websocket.ClosePolicyViolation, "hello required")
		_ = conn.Close()
		return
	}
	conn.SetReadDeadline(time.Now().Add(readDeadline))

	client := Client{ID: hello.UserID, Socket: conn}
	app.locationManager.register <- client

	go pingLoopLocation(app.locationManager, conn, hello.UserID)
	go app.handleLocationMessages(conn, hello.UserID)
}

func pingLoopLocation(lm *LocationManager, conn *websocket.Conn, uid int) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for range t.C {
		_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			_ = writeClose(conn, websocket.CloseGoingAway, "ping error")
			lm.unregister <- unreg{userID: uid, conn: conn}
			return
		}
	}
}

func (app *application) handleLocationMessages(conn *websocket.Conn, userID int) {
	defer func() {
		app.locationManager.unregister <- unreg{userID: userID, conn: conn}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = app.locationRepo.ClearLocation(ctx, userID)
		cancel()
		_ = conn.Close()
	}()

	for {
		var msg struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("location read error:", err)
			_ = writeClose(conn, websocket.CloseNormalClosure, "read error")
			return
		}

		latStr := msg.Latitude
		lonStr := msg.Longitude
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := app.locationRepo.SetLocation(ctx, models.Location{UserID: userID, Latitude: &latStr, Longitude: &lonStr})
		cancel()
		if err != nil {
			log.Println("update location error:", err)
			continue
		}

		app.locationManager.broadcast <- models.Location{UserID: userID, Latitude: &latStr, Longitude: &lonStr}
	}
}

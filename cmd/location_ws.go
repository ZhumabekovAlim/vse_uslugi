package main

import (
	"context"
	"encoding/json"
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

type locationMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type locationResponse struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
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
			msg := locationResponse{Type: "location_update", Payload: loc}
			for id, conn := range lm.clients {
				_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
				if err := conn.WriteJSON(msg); err != nil {
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
		if err := app.locationService.GoOffline(ctx, userID); err != nil {
			log.Println("go offline error:", err)
		}
		cancel()
		app.locationManager.broadcast <- models.Location{UserID: userID}
		_ = conn.Close()
	}()

	for {
		var msg locationMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("location read error:", err)
			_ = writeClose(conn, websocket.CloseNormalClosure, "read error")
			return
		}

		switch msg.Type {
		case "update_location":
			var coords struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}
			if err := json.Unmarshal(msg.Payload, &coords); err != nil {
				respondLocationError(conn, "invalid update payload")
				continue
			}

			latVal := coords.Latitude
			lonVal := coords.Longitude

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := app.locationService.SetLocation(ctx, models.Location{UserID: userID, Latitude: &latVal, Longitude: &lonVal})
			cancel()
			if err != nil {
				log.Println("update location error:", err)
				respondLocationError(conn, "failed to update location")
				continue
			}

			app.locationManager.broadcast <- models.Location{UserID: userID, Latitude: &latVal, Longitude: &lonVal}
			_ = sendLocationResponse(conn, locationResponse{Type: "location_ack"})

		case "request_executors":
			var filter models.ExecutorLocationFilter
			if len(msg.Payload) > 0 {
				if err := json.Unmarshal(msg.Payload, &filter); err != nil {
					respondLocationError(conn, "invalid filter payload")
					continue
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			execs, err := app.locationService.GetExecutors(ctx, filter)
			cancel()
			if err != nil {
				log.Println("get executors error:", err)
				respondLocationError(conn, "failed to load executors")
				continue
			}

			_ = sendLocationResponse(conn, locationResponse{Type: "executor_locations", Payload: execs})

		case "request_location":
			var payload struct {
				UserID int `json:"user_id"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.UserID == 0 {
				respondLocationError(conn, "invalid location request")
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			loc, err := app.locationService.GetLocation(ctx, payload.UserID)
			cancel()
			if err != nil {
				log.Println("get location error:", err)
				respondLocationError(conn, "failed to get location")
				continue
			}

			_ = sendLocationResponse(conn, locationResponse{Type: "user_location", Payload: loc})

		case "go_offline":
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := app.locationService.GoOffline(ctx, userID); err != nil {
				cancel()
				log.Println("go offline error:", err)
				respondLocationError(conn, "failed to go offline")
				continue
			}
			cancel()
			app.locationManager.broadcast <- models.Location{UserID: userID}
			_ = sendLocationResponse(conn, locationResponse{Type: "offline_ack"})

		default:
			respondLocationError(conn, "unknown message type")
		}
	}
}

func respondLocationError(conn *websocket.Conn, message string) {
	_ = sendLocationResponse(conn, locationResponse{Type: "error", Error: message})
}

func sendLocationResponse(conn *websocket.Conn, resp locationResponse) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	return conn.WriteJSON(resp)
}

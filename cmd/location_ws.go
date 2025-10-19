package main

import (
	"context"
	"encoding/json"
	"log"
	"naimuBack/internal/models"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultExecutorInterval = 5 * time.Second
	minExecutorInterval     = 2 * time.Second
)

// LocationManager manages websocket connections for location sharing.
type LocationManager struct {
	clients    map[int]*locationClient
	register   chan *locationClient
	unregister chan *locationClient
	broadcast  chan models.Location
}

type locationMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type locationResponse struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type locationClient struct {
	id            int
	conn          *websocket.Conn
	send          chan locationResponse
	closed        chan struct{}
	closeOnce     sync.Once
	mu            sync.RWMutex
	subscriptions map[string]*executorSubscription
}

type executorSubscription struct {
	subscriptionID string
	requestID      string
	filter         models.ExecutorLocationFilter
	ticker         *time.Ticker
	stop           chan struct{}
}

type executorSnapshot struct {
	SubscriptionID string                         `json:"subscription_id"`
	GeneratedAt    time.Time                      `json:"generated_at"`
	Executors      []models.ExecutorLocationGroup `json:"executors"`
}

func newLocationClient(id int, conn *websocket.Conn) *locationClient {
	return &locationClient{
		id:            id,
		conn:          conn,
		send:          make(chan locationResponse, 32),
		closed:        make(chan struct{}),
		subscriptions: make(map[string]*executorSubscription),
	}
}

// NewLocationManager creates a new LocationManager instance.
func NewLocationManager() *LocationManager {
	return &LocationManager{
		clients:    make(map[int]*locationClient),
		register:   make(chan *locationClient),
		unregister: make(chan *locationClient),
		broadcast:  make(chan models.Location),
	}
}

func (c *locationClient) writePump() {
	defer c.close()
	for resp := range c.send {
		if err := c.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
			log.Println("location write deadline error:", err)
		}
		if err := c.conn.WriteJSON(resp); err != nil {
			log.Println("location write error:", err)
			return
		}
	}
}

func (c *locationClient) enqueue(resp locationResponse) bool {
	select {
	case c.send <- resp:
		return true
	case <-c.closed:
		return false
	default:
		c.close()
		return false
	}
}

func (c *locationClient) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.stopAllSubscriptions()
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *locationClient) addSubscription(sub *executorSubscription) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[sub.subscriptionID] = sub
}

func (c *locationClient) removeSubscription(id string) (*executorSubscription, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subscriptions[id]
	if ok {
		delete(c.subscriptions, id)
	}
	return sub, ok
}

func (c *locationClient) stopAllSubscriptions() {
	c.mu.Lock()
	subs := make([]*executorSubscription, 0, len(c.subscriptions))
	for _, sub := range c.subscriptions {
		subs = append(subs, sub)
	}
	c.subscriptions = make(map[string]*executorSubscription)
	c.mu.Unlock()

	for _, sub := range subs {
		sub.stopSubscription()
	}
}

func (sub *executorSubscription) stopSubscription() {
	if sub == nil {
		return
	}
	if sub.ticker != nil {
		sub.ticker.Stop()
	}
	select {
	case <-sub.stop:
	default:
		close(sub.stop)
	}
}

// Run starts the manager loop.
func (lm *LocationManager) Run() {
	for {
		select {
		case client := <-lm.register:
			if old, ok := lm.clients[client.id]; ok && old != client {
				old.close()
			}
			lm.clients[client.id] = client
		case client := <-lm.unregister:
			if current, ok := lm.clients[client.id]; ok && current == client {
				current.close()
				delete(lm.clients, client.id)
			}
		case loc := <-lm.broadcast:
			msg := locationResponse{Type: "location_update", Payload: loc}
			for id, client := range lm.clients {
				if !client.enqueue(msg) {
					client.close()
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

	client := newLocationClient(hello.UserID, conn)
	go client.writePump()

	app.locationManager.register <- client

	go pingLoopLocation(app.locationManager, client)
	go app.handleLocationMessages(client)
}

func pingLoopLocation(lm *LocationManager, client *locationClient) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = writeClose(client.conn, websocket.CloseGoingAway, "ping error")
				lm.unregister <- client
				return
			}
		case <-client.closed:
			return
		}
	}
}

func (app *application) handleLocationMessages(client *locationClient) {
	conn := client.conn
	userID := client.id

	defer func() {
		app.locationManager.unregister <- client
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := app.locationService.GoOffline(ctx, userID); err != nil {
			log.Println("go offline error:", err)
		}
		cancel()
		app.locationManager.broadcast <- models.Location{UserID: userID}
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
				respondLocationError(client, msg.RequestID, "invalid update payload")
				continue
			}

			latVal := coords.Latitude
			lonVal := coords.Longitude

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := app.locationService.SetLocation(ctx, models.Location{UserID: userID, Latitude: &latVal, Longitude: &lonVal})
			cancel()
			if err != nil {
				log.Println("update location error:", err)
				respondLocationError(client, msg.RequestID, "failed to update location")
				continue
			}

			app.locationManager.broadcast <- models.Location{UserID: userID, Latitude: &latVal, Longitude: &lonVal}
			sendLocationResponse(client, locationResponse{Type: "location_ack", RequestID: msg.RequestID})

		case "request_executors":
			var filter models.ExecutorLocationFilter
			if len(msg.Payload) > 0 {
				if err := json.Unmarshal(msg.Payload, &filter); err != nil {
					respondLocationError(client, msg.RequestID, "invalid filter payload")
					continue
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			execs, err := app.locationService.GetExecutors(ctx, filter)
			cancel()
			if err != nil {
				log.Println("get executors error:", err)
				respondLocationError(client, msg.RequestID, "failed to load executors")
				continue
			}

			sendLocationResponse(client, locationResponse{Type: "executor_locations", RequestID: msg.RequestID, Payload: execs})

		case "request_location":
			var payload struct {
				UserID int `json:"user_id"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.UserID == 0 {
				respondLocationError(client, msg.RequestID, "invalid location request")
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			loc, err := app.locationService.GetLocation(ctx, payload.UserID)
			cancel()
			if err != nil {
				log.Println("get location error:", err)
				respondLocationError(client, msg.RequestID, "failed to get location")
				continue
			}

			sendLocationResponse(client, locationResponse{Type: "user_location", RequestID: msg.RequestID, Payload: loc})

		case "subscribe_executors":
			app.handleSubscribeExecutors(client, msg)

		case "unsubscribe_executors":
			app.handleUnsubscribeExecutors(client, msg)

		case "go_offline":
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := app.locationService.GoOffline(ctx, userID); err != nil {
				cancel()
				log.Println("go offline error:", err)
				respondLocationError(client, msg.RequestID, "failed to go offline")
				continue
			}
			cancel()
			app.locationManager.broadcast <- models.Location{UserID: userID}
			sendLocationResponse(client, locationResponse{Type: "offline_ack", RequestID: msg.RequestID})

		default:
			respondLocationError(client, msg.RequestID, "unknown message type")
		}
	}
}

func (app *application) handleSubscribeExecutors(client *locationClient, msg locationMessage) {
	var payload struct {
		Filter     models.ExecutorLocationFilter `json:"filter"`
		IntervalMs int64                         `json:"interval_ms,omitempty"`
	}
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			respondLocationError(client, msg.RequestID, "invalid subscription payload")
			return
		}
	}

	interval := time.Duration(payload.IntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = defaultExecutorInterval
	} else if interval < minExecutorInterval {
		interval = minExecutorInterval
	}

	subID := uuid.NewString()
	requestID := msg.RequestID
	if requestID == "" {
		requestID = subID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	execs, err := app.locationService.GetExecutors(ctx, payload.Filter)
	cancel()
	if err != nil {
		log.Println("initial executor snapshot error:", err)
		respondLocationError(client, msg.RequestID, "failed to load executors")
		return
	}

	sub := &executorSubscription{
		subscriptionID: subID,
		requestID:      requestID,
		filter:         payload.Filter,
		ticker:         time.NewTicker(interval),
		stop:           make(chan struct{}),
	}

	client.addSubscription(sub)

	ackPayload := map[string]interface{}{
		"subscription_id": subID,
		"interval_ms":     interval.Milliseconds(),
	}
	if !sendLocationResponse(client, locationResponse{Type: "subscription_ack", RequestID: msg.RequestID, Payload: ackPayload}) {
		if removed, ok := client.removeSubscription(subID); ok {
			removed.stopSubscription()
		} else {
			sub.stopSubscription()
		}
		return
	}

	snapshot := executorSnapshot{
		SubscriptionID: subID,
		GeneratedAt:    time.Now().UTC(),
		Executors:      execs,
	}
	if !sendLocationResponse(client, locationResponse{Type: "executor_snapshot", RequestID: requestID, Payload: snapshot}) {
		if removed, ok := client.removeSubscription(subID); ok {
			removed.stopSubscription()
		} else {
			sub.stopSubscription()
		}
		return
	}

	go app.runExecutorSubscription(client, sub)
}

func (app *application) handleUnsubscribeExecutors(client *locationClient, msg locationMessage) {
	var payload struct {
		SubscriptionID string `json:"subscription_id"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.SubscriptionID == "" {
		respondLocationError(client, msg.RequestID, "invalid unsubscribe payload")
		return
	}

	if sub, ok := client.removeSubscription(payload.SubscriptionID); ok {
		sub.stopSubscription()
		sendLocationResponse(client, locationResponse{Type: "unsubscribe_ack", RequestID: msg.RequestID, Payload: map[string]string{
			"subscription_id": payload.SubscriptionID,
		}})
	} else {
		respondLocationError(client, msg.RequestID, "subscription not found")
	}
}

func (app *application) runExecutorSubscription(client *locationClient, sub *executorSubscription) {
	for {
		select {
		case <-sub.ticker.C:
			if !app.pushExecutorSnapshot(client, sub) {
				if removed, ok := client.removeSubscription(sub.subscriptionID); ok {
					removed.stopSubscription()
				} else {
					sub.stopSubscription()
				}
				return
			}
		case <-sub.stop:
			return
		case <-client.closed:
			return
		}
	}
}

func (app *application) pushExecutorSnapshot(client *locationClient, sub *executorSubscription) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	execs, err := app.locationService.GetExecutors(ctx, sub.filter)
	cancel()
	if err != nil {
		log.Println("executor subscription refresh error:", err)
		respondLocationError(client, sub.requestID, "failed to refresh executors")
		return true
	}

	snapshot := executorSnapshot{
		SubscriptionID: sub.subscriptionID,
		GeneratedAt:    time.Now().UTC(),
		Executors:      execs,
	}

	return sendLocationResponse(client, locationResponse{Type: "executor_snapshot", RequestID: sub.requestID, Payload: snapshot})
}

func respondLocationError(client *locationClient, requestID, message string) {
	sendLocationResponse(client, locationResponse{Type: "error", RequestID: requestID, Error: message})
}

func sendLocationResponse(client *locationClient, resp locationResponse) bool {
	if client == nil {
		return false
	}
	return client.enqueue(resp)
}

func respondLocationError(conn *websocket.Conn, message string) {
	_ = sendLocationResponse(conn, locationResponse{Type: "error", Error: message})
}

func sendLocationResponse(conn *websocket.Conn, resp locationResponse) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	return conn.WriteJSON(resp)
}

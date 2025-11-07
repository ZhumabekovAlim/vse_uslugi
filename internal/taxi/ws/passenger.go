package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// PassengerEvent is message for passengers.
type PassengerEvent struct {
	Type     string `json:"type"`
	OrderID  int64  `json:"order_id"`
	Status   string `json:"status,omitempty"`
	Radius   int    `json:"radius,omitempty"`
	Message  string `json:"message,omitempty"`
	DriverID int64  `json:"driver_id,omitempty"`
	Price    int    `json:"price,omitempty"`
}

// PassengerHub manages passenger WS connections.
type PassengerHub struct {
	upgrader websocket.Upgrader
	logger   Logger

	mu    sync.RWMutex
	conns map[int64]*websocket.Conn
	wmu   map[int64]*sync.Mutex
}

// NewPassengerHub constructs passenger hub.
func NewPassengerHub(logger Logger) *PassengerHub {
	return &PassengerHub{
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   logger,
		conns:    make(map[int64]*websocket.Conn),
		wmu:      make(map[int64]*sync.Mutex),
	}
}

// ServeWS handles passenger connections.
func (h *PassengerHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	passengerID, err := parseIDParam(r, "passenger_id")
	if err != nil {
		http.Error(w, "missing passenger_id", http.StatusUnauthorized)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("passenger ws upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	if old, ok := h.conns[passengerID]; ok {
		_ = old.Close() // <- важный момент: закрываем старое соединение
	}
	h.conns[passengerID] = conn
	if _, ok := h.wmu[passengerID]; !ok {
		h.wmu[passengerID] = &sync.Mutex{}
	}
	h.mu.Unlock()

	go h.readLoop(passengerID, conn)
}

func (h *PassengerHub) readLoop(passengerID int64, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		h.mu.Lock()
		delete(h.conns, passengerID)
		delete(h.wmu, passengerID)
		h.mu.Unlock()
	}()

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Обрабатываем текстовый "ping" как в DriverHub
		if mt == websocket.TextMessage {
			trimmed := strings.TrimSpace(string(msg))
			if strings.EqualFold(trimmed, "ping") {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			}
		}
	}
}

func (h *PassengerHub) safeWrite(passengerID int64, writer func(*websocket.Conn) error) {
	h.mu.RLock()
	conn := h.conns[passengerID]
	mu := h.wmu[passengerID]
	h.mu.RUnlock()
	if conn == nil || mu == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := writer(conn); err != nil {
		h.logger.Errorf("passenger %d write failed: %v", passengerID, err)
	}
}

// PushOrderEvent sends event to passenger.
func (h *PassengerHub) PushOrderEvent(passengerID int64, event PassengerEvent) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	_, ok := h.conns[passengerID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	// ДОБАВЬ ЭТО:
	if h.logger != nil {
		h.logger.Infof("WS → passenger %d: %s", passengerID, string(eventBytes))
	}

	h.safeWrite(passengerID, func(conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, eventBytes)
	})
}

// BroadcastEvent sends the same payload to all connected passengers.
func (h *PassengerHub) BroadcastEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// копим список получателей под RLock
	h.mu.RLock()
	ids := make([]int64, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	h.mu.RUnlock()

	for _, id := range ids {
		h.safeWrite(id, func(conn *websocket.Conn) error {
			return conn.WriteMessage(websocket.TextMessage, data)
		})
	}
}

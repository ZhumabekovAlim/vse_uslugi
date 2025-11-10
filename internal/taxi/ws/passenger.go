package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 20 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
)

// PassengerEvent is message for passengers.
type PassengerEvent struct {
	Type     string           `json:"type"`
	OrderID  int64            `json:"order_id"`
	Status   string           `json:"status,omitempty"`
	Radius   int              `json:"radius,omitempty"`
	Message  string           `json:"message,omitempty"`
	DriverID int64            `json:"driver_id,omitempty"`
	Price    int              `json:"price,omitempty"`
	Driver   *PassengerDriver `json:"driver,omitempty"`
}

// PassengerDriver describes driver card sent to passengers with offer events.
type PassengerDriver struct {
	ID            int64   `json:"id"`
	Status        string  `json:"status"`
	CarModel      string  `json:"car_model,omitempty"`
	CarColor      string  `json:"car_color,omitempty"`
	CarNumber     string  `json:"car_number,omitempty"`
	DriverPhoto   string  `json:"driver_photo,omitempty"`
	Phone         string  `json:"phone,omitempty"`
	Rating        float64 `json:"rating"`
	CarPhotoFront string  `json:"car_photo_front,omitempty"`
	CarPhotoBack  string  `json:"car_photo_back,omitempty"`
	CarPhotoLeft  string  `json:"car_photo_left,omitempty"`
	CarPhotoRight string  `json:"car_photo_right,omitempty"`
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

	go func(id int64, conn *websocket.Conn) {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {

			h.mu.RLock()
			alive := h.conns[id] == conn
			h.mu.RUnlock()
			if !alive {
				return
			}

			h.safeWrite(id, func(c *websocket.Conn) error {
				c.SetWriteDeadline(time.Now().Add(writeWait))
				return c.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
			})
		}
	}(passengerID, conn)

	go h.readLoop(passengerID, conn)
}

func (h *PassengerHub) readLoop(passengerID int64, conn *websocket.Conn) {
	defer func() { h.closeConn(passengerID, conn) }()
	conn.SetReadLimit(16 << 10) // 16KB
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		h.logger.Infof("passenger %d closed ws (%d: %s)", passengerID, code, text)
		h.closeConn(passengerID, conn)
		return nil
	})

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(pongWait))
		if mt == websocket.TextMessage && strings.EqualFold(strings.TrimSpace(string(msg)), "ping") {
			// optional: отвечаем для совместимости
			_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
		}
	}
}

func (h *PassengerHub) closeConn(id int64, c *websocket.Conn) {
	_ = c.Close()
	h.mu.Lock()
	delete(h.conns, id)
	delete(h.wmu, id)
	h.mu.Unlock()
	if h.logger != nil {
		h.logger.Infof("🔌 closed ws passenger=%d", id)
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
		h.closeConn(passengerID, conn) // 👈 добавить
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

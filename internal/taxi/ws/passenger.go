package ws

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

// PassengerEvent is message for passengers.
type PassengerEvent struct {
    Type    string `json:"type"`
    OrderID int64  `json:"order_id"`
    Status  string `json:"status,omitempty"`
    Radius  int    `json:"radius,omitempty"`
    Message string `json:"message,omitempty"`
}

// PassengerHub manages passenger WS connections.
type PassengerHub struct {
    upgrader websocket.Upgrader
    logger   Logger

    mu    sync.RWMutex
    conns map[int64]*websocket.Conn
}

// NewPassengerHub constructs passenger hub.
func NewPassengerHub(logger Logger) *PassengerHub {
    return &PassengerHub{
        upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
        logger:   logger,
        conns:    make(map[int64]*websocket.Conn),
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
    h.conns[passengerID] = conn
    h.mu.Unlock()

    go h.readLoop(passengerID, conn)
}

func (h *PassengerHub) readLoop(passengerID int64, conn *websocket.Conn) {
    defer func() {
        conn.Close()
        h.mu.Lock()
        delete(h.conns, passengerID)
        h.mu.Unlock()
    }()

    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        if _, _, err := conn.NextReader(); err != nil {
            return
        }
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    }
}

// PushOrderEvent sends event to passenger.
func (h *PassengerHub) PushOrderEvent(passengerID int64, event PassengerEvent) {
    eventBytes, err := json.Marshal(event)
    if err != nil {
        return
    }
    h.mu.RLock()
    conn := h.conns[passengerID]
    h.mu.RUnlock()
    if conn == nil {
        return
    }
    conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
    if err := conn.WriteMessage(websocket.TextMessage, eventBytes); err != nil {
        h.logger.Errorf("passenger send failed: %v", err)
    }
}

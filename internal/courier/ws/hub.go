package ws

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// Logger defines minimal logging interface required by hubs.
type Logger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

type baseHub struct {
	name   string
	param  string
	logger Logger

	upgrader websocket.Upgrader

	mu    sync.RWMutex
	conns map[int64]*websocket.Conn
	locks map[int64]*sync.Mutex
}

func newBaseHub(name, param string, logger Logger) *baseHub {
	return &baseHub{
		name:   name,
		param:  param,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		conns: make(map[int64]*websocket.Conn),
		locks: make(map[int64]*sync.Mutex),
	}
}

func (h *baseHub) serveWS(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, h.param)
	if err != nil || id == 0 {
		http.Error(w, "missing "+h.param, http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier %s ws upgrade failed: %v", h.name, err)
		}
		return
	}

	h.mu.Lock()
	if old, ok := h.conns[id]; ok {
		_ = old.Close()
	}
	h.conns[id] = conn
	if _, ok := h.locks[id]; !ok {
		h.locks[id] = &sync.Mutex{}
	}
	h.mu.Unlock()

	if h.logger != nil {
		h.logger.Infof("courier %s %d connected", h.name, id)
	}

	go h.pingLoop(id, conn)
	go h.readLoop(id, conn)
}

func (h *baseHub) pingLoop(id int64, conn *websocket.Conn) {
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
			return c.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
		})
	}
}

func (h *baseHub) readLoop(id int64, conn *websocket.Conn) {
	defer h.closeConn(id, conn)

	conn.SetReadLimit(16 << 10)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		if h.logger != nil {
			h.logger.Infof("courier %s %d closed ws (%d: %s)", h.name, id, code, text)
		}
		h.closeConn(id, conn)
		return nil
	})

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(pongWait))
		if mt == websocket.TextMessage {
			trimmed := strings.TrimSpace(string(message))
			if strings.EqualFold(trimmed, "ping") {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			}
		}
	}
}

func (h *baseHub) closeConn(id int64, conn *websocket.Conn) {
	_ = conn.Close()
	h.mu.Lock()
	if current, ok := h.conns[id]; ok && current == conn {
		delete(h.conns, id)
		delete(h.locks, id)
	}
	h.mu.Unlock()
}

func (h *baseHub) safeWrite(id int64, fn func(*websocket.Conn) error) {
	h.mu.RLock()
	conn := h.conns[id]
	mu := h.locks[id]
	h.mu.RUnlock()
	if conn == nil || mu == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := fn(conn); err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier %s %d write failed: %v", h.name, id, err)
		}
		h.closeConn(id, conn)
	}
}

func (h *baseHub) push(id int64, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier %s marshal failed: %v", h.name, err)
		}
		return
	}
	h.safeWrite(id, func(conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, data)
	})
}

func (h *baseHub) broadcast(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier %s marshal failed: %v", h.name, err)
		}
		return
	}
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

// CourierHub manages websocket connections for couriers.
type CourierHub struct {
	*baseHub
}

// NewCourierHub constructs courier hub.
func NewCourierHub(logger Logger) *CourierHub {
	return &CourierHub{newBaseHub("courier", "courier_id", logger)}
}

// ServeWS handles courier websocket requests.
func (h *CourierHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.baseHub.serveWS(w, r)
}

// Push sends a payload to specific courier connection.
func (h *CourierHub) Push(courierID int64, payload interface{}) {
	h.baseHub.push(courierID, payload)
}

// Broadcast sends payload to all connected couriers.
func (h *CourierHub) Broadcast(payload interface{}) {
	h.baseHub.broadcast(payload)
}

// SenderHub manages websocket connections for senders.
type SenderHub struct {
	*baseHub
}

// NewSenderHub constructs sender hub.
func NewSenderHub(logger Logger) *SenderHub {
	return &SenderHub{newBaseHub("sender", "sender_id", logger)}
}

// ServeWS handles sender websocket requests.
func (h *SenderHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.baseHub.serveWS(w, r)
}

// Push sends payload to a sender connection.
func (h *SenderHub) Push(senderID int64, payload interface{}) {
	h.baseHub.push(senderID, payload)
}

// Broadcast sends payload to all connected senders.
func (h *SenderHub) Broadcast(payload interface{}) {
	h.baseHub.broadcast(payload)
}

func parseIDParam(r *http.Request, name string) (int64, error) {
	if v := r.URL.Query().Get(name); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	hyphen := strings.ReplaceAll(name, "_", "-")
	variants := []string{
		"X-" + hyphen,
		"X-" + strings.ToUpper(hyphen),
		"X-" + strings.ToLower(hyphen),
	}
	seen := make(map[string]struct{}, len(variants))
	for _, key := range variants {
		canonical := http.CanonicalHeaderKey(key)
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		if v := r.Header.Get(canonical); v != "" {
			return strconv.ParseInt(v, 10, 64)
		}
	}
	return 0, strconv.ErrSyntax
}

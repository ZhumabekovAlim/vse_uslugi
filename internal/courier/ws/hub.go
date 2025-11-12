package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"naimuBack/internal/courier/geo"
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

// CourierRoutePoint describes a waypoint for courier offers.
type CourierRoutePoint struct {
	Seq     int     `json:"seq"`
	Address string  `json:"address"`
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
}

// CourierOfferPayload represents an order offer delivered to a courier.
type CourierOfferPayload struct {
	Type             string              `json:"type"`
	OrderID          int64               `json:"order_id"`
	ClientPrice      int                 `json:"client_price"`
	RecommendedPrice int                 `json:"recommended_price"`
	DistanceM        int                 `json:"distance_m"`
	EtaSeconds       int                 `json:"eta_s"`
	ExpiresInSec     int                 `json:"expires_in"`
	Points           []CourierRoutePoint `json:"points"`
}

type courierOfferClosedPayload struct {
	Type    string `json:"type"`
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

// CourierHub manages websocket connections for couriers including location updates.
type CourierHub struct {
	upgrader websocket.Upgrader
	locator  *geo.CourierLocator
	logger   Logger

	mu         sync.RWMutex
	conns      map[int64]*websocket.Conn
	locks      map[int64]*sync.Mutex
	cities     map[int64]string
	lastStatus map[int64]string
}

// NewCourierHub constructs courier hub.
func NewCourierHub(locator *geo.CourierLocator, logger Logger) *CourierHub {
	return &CourierHub{
		upgrader:   websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		locator:    locator,
		logger:     logger,
		conns:      make(map[int64]*websocket.Conn),
		locks:      make(map[int64]*sync.Mutex),
		cities:     make(map[int64]string),
		lastStatus: make(map[int64]string),
	}
}

// ServeWS handles courier websocket requests.
func (h *CourierHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	courierID, err := parseIDParam(r, "courier_id")
	if err != nil || courierID == 0 {
		http.Error(w, "missing courier_id", http.StatusUnauthorized)
		return
	}
	city := r.URL.Query().Get("city")
	if strings.TrimSpace(city) == "" {
		city = "astana"
	}
	city = strings.ToLower(strings.TrimSpace(city))

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier ws upgrade failed: %v", err)
		}
		return
	}

	h.mu.Lock()
	if old, ok := h.conns[courierID]; ok {
		_ = old.Close()
	}
	h.conns[courierID] = conn
	if _, ok := h.locks[courierID]; !ok {
		h.locks[courierID] = &sync.Mutex{}
	}
	h.cities[courierID] = city
	if _, ok := h.lastStatus[courierID]; !ok {
		h.lastStatus[courierID] = "free"
	}
	h.mu.Unlock()

	if h.logger != nil {
		h.logger.Infof("courier %d connected (city=%s)", courierID, city)
	}

	go h.pingLoop(courierID, conn)
	go h.readLoop(courierID, conn, city)
}

func (h *CourierHub) pingLoop(id int64, conn *websocket.Conn) {
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

func (h *CourierHub) readLoop(id int64, conn *websocket.Conn, city string) {
	defer h.closeConn(id, conn)

	conn.SetReadLimit(16 << 10)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		if h.logger != nil {
			h.logger.Infof("courier %d closed ws (%d: %s)", id, code, text)
		}
		h.closeConn(id, conn)
		return nil
	})

	type payloadRaw struct {
		Lon    interface{} `json:"lon"`
		Lat    interface{} `json:"lat"`
		Status string      `json:"status"`
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(pongWait))

		trimmed := strings.TrimSpace(string(message))
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "ping") {
			if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
				if h.logger != nil {
					h.logger.Errorf("courier %d pong failed: %v", id, err)
				}
				return
			}
			continue
		}

		dec := json.NewDecoder(strings.NewReader(trimmed))
		dec.UseNumber()
		var raw payloadRaw
		if err := dec.Decode(&raw); err != nil {
			if h.logger != nil {
				h.logger.Errorf("courier %d invalid payload: %v", id, err)
			}
			continue
		}

		lon, err := parseCoordinate(raw.Lon)
		if err != nil {
			if h.logger != nil {
				h.logger.Errorf("courier %d invalid lon %v: %v", id, raw.Lon, err)
			}
			continue
		}
		lat, err := parseCoordinate(raw.Lat)
		if err != nil {
			if h.logger != nil {
				h.logger.Errorf("courier %d invalid lat %v: %v", id, raw.Lat, err)
			}
			continue
		}

		if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
			if h.logger != nil {
				h.logger.Errorf("courier %d invalid coords lon=%.8f lat=%.8f", id, lon, lat)
			}
			continue
		}
		if math.Abs(lon) < 1e-4 && math.Abs(lat) < 1e-4 {
			if h.logger != nil {
				h.logger.Errorf("courier %d near-zero coords lon=%.8f lat=%.8f", id, lon, lat)
			}
			continue
		}

		status := strings.ToLower(strings.TrimSpace(raw.Status))
		if status == "" {
			status = "free"
		}

		h.mu.Lock()
		prev := h.lastStatus[id]
		if prev == "" {
			prev = "free"
		}
		needMove := prev != status
		h.lastStatus[id] = status
		h.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if needMove {
			if err := h.locator.MoveCourier(ctx, id, city, prev, status); err != nil {
				if h.logger != nil {
					h.logger.Errorf("courier %d MoveCourier %s→%s error: %v", id, prev, status, err)
				}
				_ = h.locator.SafeUpdateCourier(ctx, id, lon, lat, city, status)
			} else {
				if err := h.locator.SafeUpdateCourier(ctx, id, lon, lat, city, status); err != nil && h.logger != nil {
					h.logger.Errorf("courier %d SafeUpdateCourier after move error: %v", id, err)
				}
			}
		} else {
			if err := h.locator.SafeUpdateCourier(ctx, id, lon, lat, city, status); err != nil && h.logger != nil {
				h.logger.Errorf("courier %d SafeUpdateCourier error: %v", id, err)
			}
		}
		cancel()
	}
}

func (h *CourierHub) closeConn(id int64, conn *websocket.Conn) {
	_ = conn.Close()
	var city string
	h.mu.Lock()
	if current, ok := h.conns[id]; ok && current == conn {
		delete(h.conns, id)
		delete(h.locks, id)
		city = h.cities[id]
		delete(h.cities, id)
		delete(h.lastStatus, id)
	}
	h.mu.Unlock()
	if city != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = h.locator.GoOffline(ctx, id, city)
		cancel()
	}
	if h.logger != nil {
		h.logger.Infof("courier %d disconnected", id)
	}
}

func (h *CourierHub) safeWrite(id int64, fn func(*websocket.Conn) error) {
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
			h.logger.Errorf("courier %d write failed: %v", id, err)
		}
		h.closeConn(id, conn)
	}
}

// Push sends a payload to specific courier connection.
func (h *CourierHub) Push(courierID int64, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier push marshal failed: %v", err)
		}
		return
	}
	h.safeWrite(courierID, func(conn *websocket.Conn) error {
		return conn.WriteMessage(websocket.TextMessage, data)
	})
}

// Broadcast sends payload to all connected couriers.
func (h *CourierHub) Broadcast(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		if h.logger != nil {
			h.logger.Errorf("courier broadcast marshal failed: %v", err)
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

// SendOffer sends an order offer to a courier.
func (h *CourierHub) SendOffer(courierID int64, payload CourierOfferPayload) {
	payload.Type = "order_offer"
	h.safeWrite(courierID, func(conn *websocket.Conn) error {
		return conn.WriteJSON(payload)
	})
}

// NotifyOfferClosed informs couriers that an offer is no longer available.
func (h *CourierHub) NotifyOfferClosed(orderID int64, courierIDs []int64, reason string) {
	if len(courierIDs) == 0 {
		if h.logger != nil {
			h.logger.Errorf("courier notify offer closed: order=%d reason=%s, no recipients", orderID, reason)
		}
		return
	}
	payload := courierOfferClosedPayload{Type: "order_offer_closed", OrderID: orderID, Reason: reason}

	h.mu.RLock()
	ids := make([]int64, 0, len(courierIDs))
	for _, id := range courierIDs {
		if _, ok := h.conns[id]; ok {
			ids = append(ids, id)
		}
	}
	h.mu.RUnlock()

	if h.logger != nil {
		h.logger.Infof("courier notify offer closed: order=%d reason=%s recipients=%v", orderID, reason, ids)
	}

	for _, id := range ids {
		copyPayload := payload
		h.safeWrite(id, func(conn *websocket.Conn) error {
			return conn.WriteJSON(copyPayload)
		})
	}
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

func parseCoordinate(value interface{}) (float64, error) {
	switch v := value.(type) {
	case nil:
		return 0, fmt.Errorf("missing coordinate")
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, fmt.Errorf("empty coordinate string")
		}
		s = strings.ReplaceAll(s, ",", ".")
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported coordinate type %T", value)
	}
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

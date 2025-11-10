package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math" // 👈 добавь для проверки "near-zero"
	"naimuBack/internal/taxi/geo"
	"net/http"
	"strconv"
	"strings" // 👈 добавь
	"sync"
	"time"
)

// Logger is shared between hubs.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DriverRoutePoint describes a waypoint for an order offer.
type DriverRoutePoint struct {
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
	Address string  `json:"address,omitempty"`
}

// DriverOfferPayload represents an offer sent to driver over WS.
type DriverOfferPayload struct {
	Type         string             `json:"type"`
	OrderID      int64              `json:"order_id"`
	FromLon      float64            `json:"from_lon"`
	FromLat      float64            `json:"from_lat"`
	ToLon        float64            `json:"to_lon"`
	ToLat        float64            `json:"to_lat"`
	ClientPrice  int                `json:"client_price"`
	DistanceM    int                `json:"distance_m"`
	EtaSeconds   int                `json:"eta_s"`
	ExpiresInSec int                `json:"expires_in"`
	Route        []DriverRoutePoint `json:"route,omitempty"`
	Passenger    *DriverPassenger   `json:"passenger,omitempty"`
}

// DriverOfferClosedPayload notifies driver that offer is no longer available.
type DriverOfferClosedPayload struct {
	Type    string `json:"type"`
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

// DriverPriceResponsePayload notifies driver about passenger decision on custom price.
type DriverPriceResponsePayload struct {
	Type    string `json:"type"`
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
	Price   int    `json:"price,omitempty"`
}

// DriverPassenger describes passenger data included with order offers.
type DriverPassenger struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	Middlename   string     `json:"middlename,omitempty"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	CityID       *int64     `json:"city_id,omitempty"`
	YearsOfExp   *int64     `json:"years_of_exp,omitempty"`
	DocOfProof   string     `json:"doc_of_proof,omitempty"`
	ReviewRating *float64   `json:"review_rating,omitempty"`
	Role         string     `json:"role,omitempty"`
	Latitude     string     `json:"latitude,omitempty"`
	Longitude    string     `json:"longitude,omitempty"`
	AvatarPath   string     `json:"avatar_path,omitempty"`
	Skills       string     `json:"skills,omitempty"`
	IsOnline     *bool      `json:"is_online,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

// DriverHub manages driver websocket connections.
type DriverHub struct {
	upgrader websocket.Upgrader
	locator  *geo.DriverLocator
	logger   Logger

	mu         sync.RWMutex
	conns      map[int64]*websocket.Conn
	wmu        map[int64]*sync.Mutex
	cities     map[int64]string
	lastStatus map[int64]string // 👈 добавь это поле
}

// NewDriverHub creates driver hub.
func NewDriverHub(locator *geo.DriverLocator, logger Logger) *DriverHub {
	return &DriverHub{
		upgrader:   websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		locator:    locator,
		logger:     logger,
		conns:      make(map[int64]*websocket.Conn),
		wmu:        make(map[int64]*sync.Mutex),
		cities:     make(map[int64]string),
		lastStatus: make(map[int64]string), // 👈
	}
}

// ServeWS handles driver websocket connections.
func (h *DriverHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	driverID, err := parseIDParam(r, "driver_id")
	if err != nil {
		http.Error(w, "missing driver_id", http.StatusUnauthorized)
		return
	}
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "default"
	}
	city = strings.ToLower(strings.TrimSpace(city)) // 👈 нормализация

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("driver ws upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	if old, ok := h.conns[driverID]; ok {
		_ = old.Close()
	}
	h.conns[driverID] = conn
	if _, ok := h.wmu[driverID]; !ok {
		h.wmu[driverID] = &sync.Mutex{}
	}
	h.cities[driverID] = city
	if _, ok := h.lastStatus[driverID]; !ok {
		h.lastStatus[driverID] = "free"
	}
	h.mu.Unlock()

	h.logger.Infof("driver %d connected (city=%s)", driverID, city)

	go func(id int64, born *websocket.Conn) {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			// если соединились заново или закрылись — выходим
			h.mu.RLock()
			alive := h.conns[id] == born
			h.mu.RUnlock()
			if !alive {
				return
			}

			h.safeWrite(id, func(c *websocket.Conn) error {
				c.SetWriteDeadline(time.Now().Add(writeWait))
				return c.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
			})
		}
	}(driverID, conn)

	go h.readLoop(driverID, conn, city)
}

func (h *DriverHub) readLoop(driverID int64, conn *websocket.Conn, city string) {
	defer func() {
		h.closeConn(driverID, conn) // 👈 используем единый close (см. ниже 4.4)
	}()

	conn.SetReadLimit(16 << 10) // 16KB (чуть больше запаса)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		if h.logger != nil {
			h.logger.Infof("driver %d closed ws (%d: %s)", driverID, code, text)
		}
		h.closeConn(driverID, conn)
		return nil
	})

	type payloadT struct {
		Lon    float64
		Lat    float64
		Status string
	}

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
				h.logger.Errorf("driver %d pong failed: %v", driverID, err)
				h.closeConn(driverID, conn) // 👈 важный фикс
				return
			}
			continue
		}

		var raw payloadRaw
		dec := json.NewDecoder(strings.NewReader(trimmed))
		dec.UseNumber()
		if err := dec.Decode(&raw); err != nil {
			h.logger.Errorf("driver %d invalid payload: %v", driverID, err)
			continue
		}

		lon, err := parseCoordinate(raw.Lon)
		if err != nil {
			h.logger.Errorf("driver %d invalid lon %v: %v", driverID, raw.Lon, err)
			continue
		}
		lat, err := parseCoordinate(raw.Lat)
		if err != nil {
			h.logger.Errorf("driver %d invalid lat %v: %v", driverID, raw.Lat, err)
			continue
		}

		payload := payloadT{
			Lon:    lon,
			Lat:    lat,
			Status: raw.Status,
		}

		// валидация координат (защита от near-zero/мусора)
		if payload.Lon < -180 || payload.Lon > 180 || payload.Lat < -90 || payload.Lat > 90 {
			h.logger.Errorf("driver %d invalid coords lon=%.8f lat=%.8f", driverID, payload.Lon, payload.Lat)
			continue
		}
		if math.Abs(payload.Lon) < 1e-4 && math.Abs(payload.Lat) < 1e-4 {
			h.logger.Errorf("driver %d near-zero coords lon=%.8f lat=%.8f", driverID, payload.Lon, payload.Lat)
			continue
		}

		status := strings.ToLower(strings.TrimSpace(payload.Status))
		if status == "" {
			status = "free"
		}

		// если статус поменялся — корректно переносим между ключами
		h.mu.Lock()
		prev := h.lastStatus[driverID]
		if prev == "" {
			prev = "free"
		}
		needMove := (prev != status)
		h.lastStatus[driverID] = status
		h.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if needMove {
			if err := h.locator.MoveDriver(ctx, driverID, city, prev, status); err != nil {
				h.logger.Errorf("driver %d MoveDriver %s→%s error: %v", driverID, prev, status, err)
				//fallback: если coords ещё не были в prev, просто SafeUpdate в новый ключ
				_ = h.locator.SafeUpdateDriver(ctx, driverID, payload.Lon, payload.Lat, city, status)
			} else {
				// после MoveDriver можно обновить координаты, чтобы они были актуальны
				if err := h.locator.SafeUpdateDriver(ctx, driverID, payload.Lon, payload.Lat, city, status); err != nil {
					h.logger.Errorf("driver %d SafeUpdateDriver after move error: %v", driverID, err)
				}
			}
		} else {
			if err := h.locator.SafeUpdateDriver(ctx, driverID, payload.Lon, payload.Lat, city, status); err != nil {
				h.logger.Errorf("driver %d SafeUpdateDriver error: %v", driverID, err)
			}
		}
		cancel()

		// при желании включай отладочный дамп (но не на каждом сообщении в проде)
		// h.locator.DebugDumpFree(context.Background(), city)
	}
}

func (h *DriverHub) closeConn(id int64, c *websocket.Conn) {
	_ = c.Close()
	h.mu.Lock()
	delete(h.conns, id)
	delete(h.wmu, id)
	delete(h.cities, id)
	delete(h.lastStatus, id)
	h.mu.Unlock()
	if h.logger != nil {
		h.logger.Infof("🔌 closed ws driver=%d", id)
	}
}

// SendOffer sends an order offer to a driver.
func (h *DriverHub) SendOffer(driverID int64, payload DriverOfferPayload) {
	payload.Type = "order_offer"
	h.safeWrite(driverID, func(c *websocket.Conn) error {
		return c.WriteJSON(payload)
	})
}

// NotifyOfferClosed informs specific drivers that the offer is no longer available.
func (h *DriverHub) NotifyOfferClosed(orderID int64, driverIDs []int64, reason string) {
	if len(driverIDs) == 0 {
		h.logger.Errorf("NotifyOfferClosed: order=%d reason=%s, driverIDs is EMPTY", orderID, reason)
		return
	}
	payload := DriverOfferClosedPayload{Type: "order_offer_closed", OrderID: orderID, Reason: reason}

	// собрать получателей под RLock как сейчас — ок
	h.mu.RLock()
	ids := make([]int64, 0, len(driverIDs))
	for _, id := range driverIDs {
		if _, ok := h.conns[id]; ok {
			ids = append(ids, id)
		}
	}
	h.mu.RUnlock()

	h.logger.Infof("NotifyOfferClosed: order=%d reason=%s, driverIDs=%v, recipients=%d",
		orderID, reason, driverIDs, len(ids))

	for _, id := range ids {
		pid := payload // копия на случай гонок
		h.safeWrite(id, func(c *websocket.Conn) error {
			return c.WriteJSON(pid)
		})
		h.logger.Infof("NotifyOfferClosed: sent to driver %d", id)
	}
}

// NotifyPriceResponse informs driver about passenger decision on price proposal.
func (h *DriverHub) NotifyPriceResponse(driverID int64, payload DriverPriceResponsePayload) {
	payload.Type = "order_offer_price_response"
	h.safeWrite(driverID, func(c *websocket.Conn) error {
		return c.WriteJSON(payload)
	})
}

// BroadcastEvent sends the same payload to every connected driver.
func (h *DriverHub) BroadcastEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Errorf("driver broadcast marshal failed: %v", err)
		return
	}
	h.mu.RLock()
	ids := make([]int64, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	h.mu.RUnlock()
	for _, id := range ids {
		id := id
		h.safeWrite(id, func(c *websocket.Conn) error {
			c.SetWriteDeadline(time.Now().Add(writeWait))
			return c.WriteMessage(websocket.TextMessage, data)
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
	if v := r.Header.Get("X-" + name); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, strconv.ErrSyntax
}

func (h *DriverHub) safeWrite(driverID int64, writer func(*websocket.Conn) error) {
	h.mu.RLock()
	conn := h.conns[driverID]
	mu := h.wmu[driverID]
	h.mu.RUnlock()
	if conn == nil || mu == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := writer(conn); err != nil {
		h.logger.Errorf("driver %d write failed: %v", driverID, err)
		h.closeConn(driverID, conn) // 👈 критично
	}
}

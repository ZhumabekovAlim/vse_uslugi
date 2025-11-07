package ws

import (
	"context"
	"encoding/json"
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
	h.conns[driverID] = conn
	h.cities[driverID] = city
	if _, ok := h.lastStatus[driverID]; !ok {
		h.lastStatus[driverID] = "free"
	}
	h.mu.Unlock()

	h.logger.Infof("driver %d connected (city=%s)", driverID, city)

	go h.readLoop(driverID, conn, city)
}

func (h *DriverHub) readLoop(driverID int64, conn *websocket.Conn, city string) {
	defer func() {
		conn.Close()
		h.mu.Lock()
		delete(h.conns, driverID)
		delete(h.cities, driverID)
		delete(h.lastStatus, driverID) // 👈 чистим
		h.mu.Unlock()
		h.logger.Infof("driver %d disconnected", driverID)
	}()

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(1000 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(1000 * time.Second))
		return nil
	})

	type payloadT struct {
		Lon    float64 `json:"lon"`
		Lat    float64 `json:"lat"`
		Status string  `json:"status"`
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(10000 * time.Second))

		trimmed := strings.TrimSpace(string(message))
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "ping") {
			if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
				h.logger.Errorf("driver %d pong failed: %v", driverID, err)
			}
			continue
		}

		var payload payloadT
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			h.logger.Errorf("driver %d invalid payload: %v", driverID, err)
			continue
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

// SendOffer sends an order offer to a driver.
func (h *DriverHub) SendOffer(driverID int64, payload DriverOfferPayload) {
	payload.Type = "order_offer"
	h.mu.RLock()
	conn := h.conns[driverID]
	h.mu.RUnlock()
	if conn == nil {
		return
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteJSON(payload); err != nil {
		h.logger.Errorf("send offer to driver %d failed: %v", driverID, err)
	}
}

// NotifyOfferClosed informs specific drivers that the offer is no longer available.
func (h *DriverHub) NotifyOfferClosed(orderID int64, driverIDs []int64, reason string) {
	if len(driverIDs) == 0 {
		return
	}
	payload := DriverOfferClosedPayload{Type: "order_offer_closed", OrderID: orderID, Reason: reason}

	h.mu.RLock()
	recipients := make([]struct {
		id   int64
		conn *websocket.Conn
	}, 0, len(driverIDs))
	for _, id := range driverIDs {
		if conn, ok := h.conns[id]; ok {
			recipients = append(recipients, struct {
				id   int64
				conn *websocket.Conn
			}{id: id, conn: conn})
		}
	}
	h.mu.RUnlock()

	for _, recipient := range recipients {
		recipient.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := recipient.conn.WriteJSON(payload); err != nil {
			h.logger.Errorf("notify offer closed to driver %d failed: %v", recipient.id, err)
		}
	}
}

// NotifyPriceResponse informs driver about passenger decision on price proposal.
func (h *DriverHub) NotifyPriceResponse(driverID int64, payload DriverPriceResponsePayload) {
	payload.Type = "order_offer_price_response"

	h.mu.RLock()
	conn, ok := h.conns[driverID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteJSON(payload); err != nil {
		h.logger.Errorf("notify price response to driver %d failed: %v", driverID, err)
	}
}

// BroadcastEvent sends the same payload to every connected driver.
func (h *DriverHub) BroadcastEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Errorf("driver broadcast marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	recipients := make([]struct {
		id   int64
		conn *websocket.Conn
	}, 0, len(h.conns))
	for id, conn := range h.conns {
		recipients = append(recipients, struct {
			id   int64
			conn *websocket.Conn
		}{id: id, conn: conn})
	}
	h.mu.RUnlock()

	for _, recipient := range recipients {
		recipient.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := recipient.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Errorf("driver broadcast to %d failed: %v", recipient.id, err)
		}
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

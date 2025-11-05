package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"naimuBack/internal/taxi/geo"
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
}

// DriverHub manages driver websocket connections.
type DriverHub struct {
	upgrader websocket.Upgrader
	locator  *geo.DriverLocator
	logger   Logger

	mu     sync.RWMutex
	conns  map[int64]*websocket.Conn
	cities map[int64]string
}

// NewDriverHub creates driver hub.
func NewDriverHub(locator *geo.DriverLocator, logger Logger) *DriverHub {
	return &DriverHub{
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		locator:  locator,
		logger:   logger,
		conns:    make(map[int64]*websocket.Conn),
		cities:   make(map[int64]string),
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

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("driver ws upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.conns[driverID] = conn
	h.cities[driverID] = city
	h.mu.Unlock()

	h.logger.Infof("driver %d connected", driverID)

	go h.readLoop(driverID, conn, city)
}

func (h *DriverHub) readLoop(driverID int64, conn *websocket.Conn, city string) {
	defer func() {
		conn.Close()
		h.mu.Lock()
		delete(h.conns, driverID)
		delete(h.cities, driverID)
		h.mu.Unlock()
		h.logger.Infof("driver %d disconnected", driverID)
	}()

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(1000 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(1000 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(1000 * time.Second))
		var payload struct {
			Lon    float64 `json:"lon"`
			Lat    float64 `json:"lat"`
			Status string  `json:"status"`
		}
		if err := json.Unmarshal(message, &payload); err != nil {
			h.logger.Errorf("driver %d invalid payload: %v", driverID, err)
			continue
		}
		status := payload.Status
		if status == "" {
			status = "free"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = h.locator.UpdateDriver(ctx, driverID, payload.Lon, payload.Lat, city, status)
		cancel()
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

func parseIDParam(r *http.Request, name string) (int64, error) {
	if v := r.URL.Query().Get(name); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	if v := r.Header.Get("X-" + name); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, strconv.ErrSyntax
}

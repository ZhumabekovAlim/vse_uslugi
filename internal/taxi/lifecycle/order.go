package lifecycle

import (
	"errors"
	"math"
	"time"

	"naimuBack/internal/taxi/fsm"
)

// GeoPoint describes geographic coordinates (WGS84).
type GeoPoint struct {
	Lon float64
	Lat float64
}

// DistanceTo returns distance in meters using a haversine approximation.
func (p GeoPoint) DistanceTo(other GeoPoint) float64 {
	const earthRadius = 6371000.0
	lat1 := toRadians(p.Lat)
	lat2 := toRadians(other.Lat)
	dLat := lat2 - lat1
	dLon := toRadians(other.Lon - p.Lon)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func toRadians(v float64) float64 {
	return v * math.Pi / 180
}

// Telemetry contains driver's current coordinates and metadata.
type Telemetry struct {
	Position  GeoPoint
	SpeedKPH  float64
	Timestamp time.Time
}

// WaitSessionType enumerates waiting session kinds.
type WaitSessionType string

const (
	WaitSessionFree  WaitSessionType = "free"
	WaitSessionPaid  WaitSessionType = "paid"
	WaitSessionPause WaitSessionType = "in_trip_pause"
)

// WaitSession stores waiting window details.
type WaitSession struct {
	Type     WaitSessionType
	Started  time.Time
	Finished *time.Time
	Minutes  int
	Amount   int64
}

// WaypointKind specifies the role of the waypoint.
type WaypointKind string

const (
	WaypointPickup WaypointKind = "pickup"
	WaypointStop   WaypointKind = "stop"
	WaypointFinish WaypointKind = "finish"
)

// WaypointTarget defines a stop within the planned route.
type WaypointTarget struct {
	Kind    WaypointKind
	Name    string
	Point   GeoPoint
	Radius  float64
	Comment string
}

// WaypointProgress keeps runtime waypoint status.
type WaypointProgress struct {
	WaypointTarget
	ReachedAt *time.Time
}

// StatusEvent captures status timeline.
type StatusEvent struct {
	Status string
	At     time.Time
	Note   string
}

// WaypointEvent logs reaching a waypoint.
type WaypointEvent struct {
	Index int
	Kind  WaypointKind
	Name  string
	At    time.Time
}

// ContactAttempt stores information about chat/call attempts.
type ContactAttempt struct {
	Channel string
	At      time.Time
}

// FareBreakdown keeps final check composition.
type FareBreakdown struct {
	BaseAmount          int64
	WaitingPaidAmount   int64
	WaitingPauseAmount  int64
	ExtraDistanceAmount int64
	DiscountAmount      int64
	FreeWaitingMinutes  int
	PaidWaitingMinutes  int
	PauseWaitingMinutes int
	ExtraDistanceMeters int
}

// Total returns the payable amount considering discounts.
func (f FareBreakdown) Total() int64 {
	return f.BaseAmount + f.WaitingPaidAmount + f.WaitingPauseAmount + f.ExtraDistanceAmount - f.DiscountAmount
}

// Order aggregates runtime information about a taxi ride lifecycle.
type Order struct {
	ID          int64
	PassengerID int64
	DriverID    int64
	Status      string
	BaseFare    int64
	Currency    string

	CreatedAt time.Time
	UpdatedAt time.Time

	Waypoints      []WaypointProgress
	nextWaypoint   int
	Timeline       []StatusEvent
	WaypointLog    []WaypointEvent
	Waiting        []WaitSession
	activeWaitIdx  int
	Fare           FareBreakdown
	ContactHistory []ContactAttempt

	ArrivedAt          *time.Time
	StartedAt          *time.Time
	FinishedAt         *time.Time
	PaymentConfirmed   *time.Time
	OfferExpiresAt     time.Time
	lastKnownTelemetry Telemetry

	buttonState map[Action]*buttonState
}

// ErrInvalidOperation is returned when an action cannot be performed.
var ErrInvalidOperation = errors.New("invalid operation for current state")

// ErrOutdatedTelemetry indicates GPS data is stale.
var ErrOutdatedTelemetry = errors.New("telemetry is outdated")

// ErrGeoConstraintViolation indicates that geo validation failed.
var ErrGeoConstraintViolation = errors.New("geo constraint violation")

// ErrPinRequired indicates that a PIN is required.
var ErrPinRequired = errors.New("pin confirmation required")

type buttonState struct {
	count     int
	lastPress time.Time
	expiresAt time.Time
}

// NewOrder constructs a runtime order aggregate.
func NewOrder(id, passengerID, driverID int64, baseFare int64, currency string, createdAt time.Time, offerTTL time.Duration, route []WaypointTarget) (*Order, error) {
	if len(route) < 2 {
		return nil, errors.New("route must contain at least pickup and finish waypoints")
	}
	waypoints := make([]WaypointProgress, len(route))
	for i, wp := range route {
		waypoints[i] = WaypointProgress{WaypointTarget: wp}
	}
	order := &Order{
		ID:            id,
		PassengerID:   passengerID,
		DriverID:      driverID,
		Status:        fsm.StatusAssigned,
		BaseFare:      baseFare,
		Currency:      currency,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
		Waypoints:     waypoints,
		nextWaypoint:  0,
		activeWaitIdx: -1,
		buttonState:   make(map[Action]*buttonState),
	}
	order.OfferExpiresAt = createdAt.Add(offerTTL)
	order.appendStatus(fsm.StatusAssigned, createdAt, "driver assigned")
	return order, nil
}

func (o *Order) appendStatus(status string, at time.Time, note string) {
	if o.Status == status {
		// still record status event for audit but do not duplicate timeline entry
		if len(o.Timeline) > 0 {
			last := o.Timeline[len(o.Timeline)-1]
			if last.Status == status {
				return
			}
		}
	}
	o.Status = status
	o.UpdatedAt = at
	o.Timeline = append(o.Timeline, StatusEvent{Status: status, At: at, Note: note})
}

func (o *Order) ensureWaypointReached(index int, at time.Time) {
	if index < 0 || index >= len(o.Waypoints) {
		return
	}
	if o.Waypoints[index].ReachedAt == nil {
		o.Waypoints[index].ReachedAt = &at
		o.WaypointLog = append(o.WaypointLog, WaypointEvent{
			Index: index,
			Kind:  o.Waypoints[index].Kind,
			Name:  o.Waypoints[index].Name,
			At:    at,
		})
	}
	if o.nextWaypoint <= index {
		o.nextWaypoint = index + 1
	}
}

func (o *Order) activeWaypoint() (*WaypointProgress, int, bool) {
	if o.nextWaypoint >= len(o.Waypoints) {
		return nil, -1, false
	}
	return &o.Waypoints[o.nextWaypoint], o.nextWaypoint, true
}

func (o *Order) startWaitingSession(t WaitSessionType, at time.Time) {
	if o.activeWaitIdx >= 0 {
		if o.Waiting[o.activeWaitIdx].Type == t {
			return
		}
	}
	o.Waiting = append(o.Waiting, WaitSession{Type: t, Started: at})
	o.activeWaitIdx = len(o.Waiting) - 1
}

func (o *Order) closeActiveWaiting(at time.Time, cfg Config) {
	if o.activeWaitIdx < 0 || o.activeWaitIdx >= len(o.Waiting) {
		return
	}
	session := &o.Waiting[o.activeWaitIdx]
	if session.Finished != nil {
		return
	}
	session.Finished = &at
	duration := at.Sub(session.Started)
	if duration < 0 {
		duration = 0
	}
	minutes := int(math.Ceil(duration.Minutes()))
	if minutes < 0 {
		minutes = 0
	}
	session.Minutes = minutes
	var rate int64
	switch session.Type {
	case WaitSessionFree:
		o.Fare.FreeWaitingMinutes += minutes
	case WaitSessionPaid:
		o.Fare.PaidWaitingMinutes += minutes
		rate = cfg.PaidWaitingRatePerMinute
		session.Amount = rate * int64(minutes)
		o.Fare.WaitingPaidAmount += session.Amount
	case WaitSessionPause:
		o.Fare.PauseWaitingMinutes += minutes
		rate = cfg.PauseRatePerMinute
		session.Amount = rate * int64(minutes)
		o.Fare.WaitingPauseAmount += session.Amount
	}
	o.activeWaitIdx = -1
}

func (o *Order) activeWaitingSession() *WaitSession {
	if o.activeWaitIdx < 0 || o.activeWaitIdx >= len(o.Waiting) {
		return nil
	}
	return &o.Waiting[o.activeWaitIdx]
}

func (o *Order) recordTelemetry(t Telemetry) {
	o.lastKnownTelemetry = t
}

// AddExtraDistance registers additional kilometers with pre-calculated amount.
func (o *Order) AddExtraDistance(meters int, amount int64) {
	if meters <= 0 && amount == 0 {
		return
	}
	if meters > 0 {
		o.Fare.ExtraDistanceMeters += meters
	}
	if amount != 0 {
		o.Fare.ExtraDistanceAmount += amount
	}
}

// ApplyDiscount registers a discount applied to the order.
func (o *Order) ApplyDiscount(amount int64) {
	if amount <= 0 {
		return
	}
	o.Fare.DiscountAmount += amount
}

// LogContactAttempt stores information about driver/passenger communication.
func (o *Order) LogContactAttempt(channel string, at time.Time) {
	o.ContactHistory = append(o.ContactHistory, ContactAttempt{Channel: channel, At: at})
}

func (o *Order) buttonStateFor(action Action) *buttonState {
	state, ok := o.buttonState[action]
	if !ok {
		state = &buttonState{}
		o.buttonState[action] = state
	}
	return state
}

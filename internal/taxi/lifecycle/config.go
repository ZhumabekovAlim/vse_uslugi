package lifecycle

import "time"

// Config aggregates behavioural parameters for the taxi order lifecycle.
type Config struct {
	// ArrivalRadiusMeters is the maximum distance to the pickup point that
	// still counts as "driver has arrived".
	ArrivalRadiusMeters float64
	// StartRadiusMeters is the maximum distance to allow trip start.
	StartRadiusMeters float64
	// WaypointRadiusMeters is the geofence radius for intermediate stops.
	WaypointRadiusMeters float64
	// FinishRadiusMeters is the allowed distance to the final point when
	// finishing the order.
	FinishRadiusMeters float64
	// StationarySpeedKPH defines the "almost not moving" speed threshold.
	StationarySpeedKPH float64
	// CoordinateFreshness is the maximum age of GPS telemetry.
	CoordinateFreshness time.Duration
	// FreeWaitingWindow is the length of the complimentary waiting window
	// at the pickup point.
	FreeWaitingWindow time.Duration
	// PaidWaitingRatePerMinute controls the paid waiting price (tenge/min).
	PaidWaitingRatePerMinute int64
	// PauseRatePerMinute controls the in-trip pause price (tenge/min).
	PauseRatePerMinute int64
	// RequireBoardingPIN toggles PIN confirmation before starting a trip.
	RequireBoardingPIN bool
	// OfferTTL defines how long the dispatch offer is valid.
	OfferTTL time.Duration
	// ButtonPolicies configures throttle/cooldown for driver's actions.
	ButtonPolicies map[Action]ButtonPolicy
}

// ButtonPolicy configures how often a specific action can be triggered.
type ButtonPolicy struct {
	// Cooldown enforces a minimal duration between two presses.
	Cooldown time.Duration
	// MaxPresses limits button presses within the TTL window. Zero means no limit.
	MaxPresses int
	// TTL defines the time window for MaxPresses accounting. Zero disables TTL logic.
	TTL time.Duration
}

// Action identifies a idempotent driver action.
type Action string

const (
	ActionArrive Action = "arrive"
	ActionStart  Action = "start"
	ActionFinish Action = "finish"
	ActionPause  Action = "pause"
	ActionResume Action = "resume"
	ActionNoShow Action = "no_show"
	ActionCancel Action = "cancel"
)

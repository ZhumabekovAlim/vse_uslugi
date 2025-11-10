package lifecycle

// List of courier lifecycle statuses.
const (
	StatusNew             = "searching"
	StatusOffered         = "offered"
	StatusAssigned        = "assigned"
	StatusCourierArrived  = "courier_arrived"
	StatusPickupStarted   = "pickup_started"
	StatusPickupDone      = "pickup_done"
	StatusDeliveryStarted = "delivery_started"
	StatusDelivered       = "delivered"
	StatusClosed          = "closed"

	StatusCanceledBySender  = "canceled_by_sender"
	StatusCanceledByCourier = "canceled_by_courier"
	StatusCanceledNoShow    = "canceled_no_show"
)

var transitions = map[string]map[string]struct{}{
	StatusNew: {
		StatusOffered:          {},
		StatusCanceledBySender: {},
	},
	StatusOffered: {
		StatusAssigned:          {},
		StatusCanceledBySender:  {},
		StatusCanceledByCourier: {},
	},
	StatusAssigned: {
		StatusCourierArrived:    {},
		StatusCanceledBySender:  {},
		StatusCanceledByCourier: {},
		StatusCanceledNoShow:    {},
	},
	StatusCourierArrived: {
		StatusPickupStarted:     {},
		StatusCanceledBySender:  {},
		StatusCanceledByCourier: {},
		StatusCanceledNoShow:    {},
	},
	StatusPickupStarted: {
		StatusPickupDone:        {},
		StatusCanceledByCourier: {},
	},
	StatusPickupDone: {
		StatusDeliveryStarted:   {},
		StatusCanceledByCourier: {},
	},
	StatusDeliveryStarted: {
		StatusDelivered:         {},
		StatusCanceledByCourier: {},
	},
	StatusDelivered: {
		StatusClosed: {},
	},
}

// CanTransition returns true when the lifecycle allows moving from current to next status.
func CanTransition(current, next string) bool {
	if current == next {
		return true
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

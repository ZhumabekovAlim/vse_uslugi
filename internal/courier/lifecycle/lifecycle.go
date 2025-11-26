package lifecycle

// List of courier lifecycle statuses.
const (
	StatusNew         = "searching"
	StatusAccepted    = "accepted"
	StatusWaitingFree = "waiting_free"
	StatusInProgress  = "in_progress"
	StatusCompleted   = "completed"
	StatusClosed      = "closed"

	StatusCanceledBySender  = "canceled_by_sender"
	StatusCanceledByCourier = "canceled_by_courier"
	StatusCanceledNoShow    = "canceled_no_show"
)

var transitions = map[string]map[string]struct{}{
	StatusNew: {
		StatusAccepted:         {},
		StatusCanceledBySender: {},
	},
	StatusAccepted: {
		StatusWaitingFree:       {},
		StatusInProgress:        {},
		StatusCompleted:         {},
		StatusCanceledBySender:  {},
		StatusCanceledByCourier: {},
		StatusCanceledNoShow:    {},
	},
	StatusWaitingFree: {
		StatusInProgress:        {},
		StatusCanceledBySender:  {},
		StatusCanceledByCourier: {},
		StatusCanceledNoShow:    {},
	},
	StatusInProgress: {
		StatusCompleted:         {},
		StatusCanceledByCourier: {},
		StatusCanceledBySender:  {},
	},
	StatusCompleted: {
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

package fsm

import (
	"context"
	"database/sql"
	"errors"
)

// Status constants used by the taxi order state machine.
const (
	StatusCreated             = "created"
	StatusSearching           = "searching"
	StatusAccepted            = "accepted"
	StatusArrived             = "arrived"
	StatusPickedUp            = "picked_up"
	StatusCompleted           = "completed"
	StatusPaid                = "paid"
	StatusClosed              = "closed"
	StatusNotFound            = "not_found"
	StatusCanceled            = "canceled"
	StatusAssigned            = "assigned"
	StatusDriverAtPickup      = "driver_at_pickup"
	StatusWaitingFree         = "waiting_free"
	StatusWaitingPaid         = "waiting_paid"
	StatusInProgress          = "in_progress"
	StatusAtLastPoint         = "at_last_point"
	StatusCanceledByPassenger = "canceled_by_passenger"
	StatusCanceledByDriver    = "canceled_by_driver"
	StatusNoShow              = "no_show"
)

var transitions = map[string]map[string]struct{}{
	StatusCreated:   {StatusSearching: {}},
	StatusSearching: {StatusAccepted: {}, StatusNotFound: {}, StatusCanceled: {}},
	StatusAccepted: {
		StatusArrived:             {},
		StatusAssigned:            {},
		StatusDriverAtPickup:      {},
		StatusWaitingFree:         {},
		StatusWaitingPaid:         {},
		StatusInProgress:          {},
		StatusCanceled:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusAssigned: {
		StatusDriverAtPickup:      {},
		StatusArrived:             {},
		StatusWaitingFree:         {},
		StatusWaitingPaid:         {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusDriverAtPickup: {
		StatusWaitingFree:         {},
		StatusWaitingPaid:         {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusWaitingFree: {
		StatusWaitingPaid:         {},
		StatusInProgress:          {},
		StatusPickedUp:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
		StatusNoShow:              {},
	},
	StatusWaitingPaid: {
		StatusInProgress:          {},
		StatusPickedUp:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
		StatusNoShow:              {},
	},
	StatusArrived: {
		StatusPickedUp:            {},
		StatusCanceled:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusPickedUp: {
		StatusCompleted:           {},
		StatusInProgress:          {},
		StatusCanceled:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusInProgress: {
		StatusAtLastPoint:         {},
		StatusCompleted:           {},
		StatusCanceled:            {},
		StatusCanceledByPassenger: {},
		StatusCanceledByDriver:    {},
	},
	StatusAtLastPoint: {
		StatusCompleted:        {},
		StatusCanceledByDriver: {},
	},
	StatusCompleted: {
		StatusPaid:   {},
		StatusClosed: {},
	},
	StatusPaid:   {StatusClosed: {}},
	StatusClosed: {},
	StatusNotFound: {
		StatusClosed: {},
	},
	StatusCanceled: {
		StatusClosed: {},
	},
	StatusCanceledByPassenger: {
		StatusClosed: {},
	},
	StatusCanceledByDriver: {
		StatusClosed: {},
	},
	StatusNoShow: {
		StatusClosed: {},
	},
}

// CanTransition returns whether the order can transition from the current status to the target status.
func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// Apply updates an order status using optimistic validation.
func Apply(ctx context.Context, tx *sql.Tx, orderID int64, fromStatus, toStatus string) error {
	if !CanTransition(fromStatus, toStatus) {
		return errors.New("invalid status transition")
	}
	res, err := tx.ExecContext(ctx, `UPDATE orders SET status = ? WHERE id = ? AND status = ?`, toStatus, orderID, fromStatus)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

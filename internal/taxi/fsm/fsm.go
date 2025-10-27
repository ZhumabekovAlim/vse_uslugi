package fsm

import (
    "context"
    "database/sql"
    "errors"
)

var transitions = map[string]map[string]struct{}{
    "created":    {"searching": {}},
    "searching":  {"accepted": {}, "not_found": {}, "canceled": {}},
    "accepted":   {"arrived": {}, "canceled": {}},
    "arrived":    {"picked_up": {}, "canceled": {}},
    "picked_up":  {"completed": {}, "canceled": {}},
    "completed":  {"paid": {}, "closed": {}},
    "paid":       {"closed": {}},
    "closed":     {},
    "not_found":  {"closed": {}},
    "canceled":   {"closed": {}},
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

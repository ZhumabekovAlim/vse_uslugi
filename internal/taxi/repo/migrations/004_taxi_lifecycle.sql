-- +migrate Up
ALTER TABLE orders
    MODIFY status ENUM(
        'created',
        'searching',
        'accepted',
        'assigned',
        'driver_at_pickup',
        'waiting_free',
        'waiting_paid',
        'in_progress',
        'at_last_point',
        'arrived',
        'picked_up',
        'completed',
        'paid',
        'closed',
        'not_found',
        'canceled',
        'canceled_by_passenger',
        'canceled_by_driver',
        'no_show'
    ) NOT NULL DEFAULT 'created';

-- Preserve compatibility for legacy statuses by ensuring new values exist even if previous ones were stored.

-- +migrate Down
ALTER TABLE orders
    MODIFY status ENUM(
        'created',
        'searching',
        'accepted',
        'arrived',
        'picked_up',
        'completed',
        'paid',
        'closed',
        'not_found',
        'canceled'
    ) NOT NULL DEFAULT 'created';

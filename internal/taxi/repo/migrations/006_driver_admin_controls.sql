-- +migrate Up
ALTER TABLE drivers
    ADD COLUMN approval_status ENUM('pending','approved','rejected') NOT NULL DEFAULT 'pending' AFTER status,
    ADD COLUMN is_banned TINYINT(1) NOT NULL DEFAULT 0 AFTER approval_status;

-- Existing drivers are considered approved to avoid service disruption
UPDATE drivers SET approval_status = 'approved';

-- +migrate Down
ALTER TABLE drivers
    DROP COLUMN is_banned,
    DROP COLUMN approval_status;

-- +migrate Up
ALTER TABLE drivers ADD COLUMN balance INT NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE drivers DROP COLUMN balance;

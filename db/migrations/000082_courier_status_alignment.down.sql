ALTER TABLE couriers
    MODIFY status VARCHAR(32) NOT NULL DEFAULT 'pending';

UPDATE couriers
SET status = CASE
        WHEN is_banned = 1 THEN 'banned'
        WHEN approval_status = 'pending' THEN 'pending'
        ELSE 'active'
    END;

ALTER TABLE couriers
    DROP COLUMN approval_status,
    DROP COLUMN is_banned;

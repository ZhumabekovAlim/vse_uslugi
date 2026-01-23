ALTER TABLE business_seat_purchases
    ADD COLUMN expires_at DATETIME NULL DEFAULT NULL AFTER payload_json;

UPDATE business_seat_purchases p
LEFT JOIN business_accounts a ON a.business_user_id = p.business_user_id
SET p.expires_at = COALESCE(a.seats_expires_at, DATE_ADD(p.created_at, INTERVAL 30 DAY))
WHERE p.expires_at IS NULL;

ALTER TABLE business_seat_purchases
    MODIFY expires_at DATETIME NOT NULL;

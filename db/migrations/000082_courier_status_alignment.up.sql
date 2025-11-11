ALTER TABLE couriers
    ADD COLUMN approval_status ENUM('pending','approved','rejected') NOT NULL DEFAULT 'pending' AFTER status,
    ADD COLUMN is_banned TINYINT(1) NOT NULL DEFAULT 0 AFTER approval_status;

UPDATE couriers
SET
    approval_status = CASE
        WHEN status = 'pending' THEN 'pending'
        WHEN status = 'banned' THEN 'approved'
        ELSE 'approved'
    END,
    is_banned = CASE WHEN status = 'banned' THEN 1 ELSE 0 END,
    status = CASE
        WHEN status IN ('offline','free','busy') THEN status
        ELSE 'offline'
    END;

ALTER TABLE couriers
    MODIFY status ENUM('offline','free','busy') NOT NULL DEFAULT 'offline';

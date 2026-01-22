ALTER TABLE rent
    MODIFY price BIGINT NOT NULL,
    DROP COLUMN price_to,
    DROP COLUMN negotiable,
    DROP COLUMN hide_phone;

use naimudb;

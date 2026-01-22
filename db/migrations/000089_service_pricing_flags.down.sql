ALTER TABLE service
    MODIFY price BIGINT NOT NULL,
    DROP COLUMN price_to,
    DROP COLUMN on_site,
    DROP COLUMN negotiable,
    DROP COLUMN hide_phone;

use naimudb;

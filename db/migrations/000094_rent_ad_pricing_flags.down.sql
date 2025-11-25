ALTER TABLE rent_ad
    MODIFY price DECIMAL(10, 2) NOT NULL,
    DROP COLUMN price_to,
    DROP COLUMN negotiable,
    DROP COLUMN hide_phone;

use naimudb;

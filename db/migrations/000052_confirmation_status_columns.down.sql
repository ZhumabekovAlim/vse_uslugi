ALTER TABLE service_confirmations
    DROP COLUMN status;

ALTER TABLE ad_confirmations
    DROP COLUMN status;

ALTER TABLE work_confirmations
    DROP COLUMN status;

ALTER TABLE work_ad_confirmations
    DROP COLUMN status;

ALTER TABLE rent_confirmations
    DROP COLUMN status;

ALTER TABLE rent_ad_confirmations
    DROP COLUMN status;

use naimudb;

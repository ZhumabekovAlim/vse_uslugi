ALTER TABLE service_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

ALTER TABLE ad_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

ALTER TABLE work_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

ALTER TABLE work_ad_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

ALTER TABLE rent_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

ALTER TABLE rent_ad_confirmations
    ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'active' AFTER confirmed;

use naimudb;

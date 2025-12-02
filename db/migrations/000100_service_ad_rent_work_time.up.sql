ALTER TABLE service
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER subcategory_id,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

ALTER TABLE ad
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER subcategory_id,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

ALTER TABLE rent
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER subcategory_id,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

ALTER TABLE rent_ad
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER subcategory_id,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

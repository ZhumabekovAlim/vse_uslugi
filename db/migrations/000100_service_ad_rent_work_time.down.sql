ALTER TABLE service
    DROP COLUMN work_time_to,
    DROP COLUMN work_time_from;

ALTER TABLE ad
    DROP COLUMN work_time_to,
    DROP COLUMN work_time_from;

ALTER TABLE rent
    DROP COLUMN work_time_to,
    DROP COLUMN work_time_from;

ALTER TABLE rent_ad
    DROP COLUMN work_time_to,
    DROP COLUMN work_time_from;

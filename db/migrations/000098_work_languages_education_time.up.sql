ALTER TABLE work
    ADD COLUMN languages JSON NOT NULL DEFAULT ('["Казахский","Русский"]') AFTER payment_period,
    ADD COLUMN education VARCHAR(255) NOT NULL DEFAULT '' AFTER languages,
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER education,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

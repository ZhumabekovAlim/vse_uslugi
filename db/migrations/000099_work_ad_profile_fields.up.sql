ALTER TABLE work_ad
    ADD COLUMN languages JSON NOT NULL DEFAULT ('["Казахский","Русский"]') AFTER payment_period,
    ADD COLUMN education VARCHAR(255) NOT NULL DEFAULT '' AFTER languages,
    ADD COLUMN first_name VARCHAR(255) NOT NULL DEFAULT '' AFTER education,
    ADD COLUMN last_name VARCHAR(255) NOT NULL DEFAULT '' AFTER first_name,
    ADD COLUMN birth_date VARCHAR(255) NOT NULL DEFAULT '' AFTER last_name,
    ADD COLUMN contact_number VARCHAR(255) NOT NULL DEFAULT '' AFTER birth_date,
    ADD COLUMN work_time_from VARCHAR(255) NOT NULL DEFAULT '' AFTER contact_number,
    ADD COLUMN work_time_to VARCHAR(255) NOT NULL DEFAULT '' AFTER work_time_from;

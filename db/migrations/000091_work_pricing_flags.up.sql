ALTER TABLE work
    MODIFY price BIGINT NULL,
    ADD COLUMN price_to BIGINT NULL AFTER price,
    ADD COLUMN negotiable BOOLEAN NOT NULL DEFAULT false AFTER price_to,
    ADD COLUMN hide_phone BOOLEAN NOT NULL DEFAULT false AFTER negotiable;

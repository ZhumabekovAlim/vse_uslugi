ALTER TABLE work
    MODIFY price DECIMAL(10, 2) NULL,
    ADD COLUMN price_to DECIMAL(10, 2) NULL AFTER price,
    ADD COLUMN negotiable BOOLEAN NOT NULL DEFAULT false AFTER price_to,
    ADD COLUMN hide_phone BOOLEAN NOT NULL DEFAULT false AFTER negotiable;

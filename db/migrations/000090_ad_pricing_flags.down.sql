UPDATE ad SET price = 0 WHERE price IS NULL;
ALTER TABLE ad
    MODIFY price BIGINT NOT NULL,
    DROP COLUMN hide_phone,
    DROP COLUMN negotiable,
    DROP COLUMN price_to,
    DROP COLUMN on_site;

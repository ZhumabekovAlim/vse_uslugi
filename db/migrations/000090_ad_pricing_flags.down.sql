UPDATE ad SET price = 0 WHERE price IS NULL;
ALTER TABLE ad
    MODIFY price DECIMAL(10, 2) NOT NULL,
    DROP COLUMN hide_phone,
    DROP COLUMN negotiable,
    DROP COLUMN price_to,
    DROP COLUMN on_site;

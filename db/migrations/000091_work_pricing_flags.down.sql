UPDATE work SET price = 0 WHERE price IS NULL;
ALTER TABLE work
    DROP COLUMN hide_phone,
    DROP COLUMN negotiable,
    DROP COLUMN price_to,
    MODIFY price DECIMAL(10, 2) NOT NULL;

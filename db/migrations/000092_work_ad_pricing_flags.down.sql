UPDATE work_ad SET price = 0 WHERE price IS NULL;
ALTER TABLE work_ad
    DROP COLUMN hide_phone,
    DROP COLUMN negotiable,
    DROP COLUMN price_to,
    MODIFY price BIGINT NOT NULL;

ALTER TABLE rent_ad
    DROP FOREIGN KEY fk_rent_ad_city_id,
    DROP COLUMN city_id;

ALTER TABLE rent
    DROP FOREIGN KEY fk_rent_city_id,
    DROP COLUMN city_id;

ALTER TABLE service
    DROP FOREIGN KEY fk_service_city_id,
    DROP COLUMN city_id;

ALTER TABLE ad
    DROP FOREIGN KEY fk_ad_city_id,
    DROP COLUMN city_id;

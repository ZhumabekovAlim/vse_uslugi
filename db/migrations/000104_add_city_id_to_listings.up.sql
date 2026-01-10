ALTER TABLE ad
    ADD COLUMN city_id INT,
    ADD CONSTRAINT fk_ad_city_id FOREIGN KEY (city_id) REFERENCES cities (id) ON DELETE CASCADE;

ALTER TABLE service
    ADD COLUMN city_id INT,
    ADD CONSTRAINT fk_service_city_id FOREIGN KEY (city_id) REFERENCES cities (id) ON DELETE CASCADE;

ALTER TABLE rent
    ADD COLUMN city_id INT,
    ADD CONSTRAINT fk_rent_city_id FOREIGN KEY (city_id) REFERENCES cities (id) ON DELETE CASCADE;

ALTER TABLE rent_ad
    ADD COLUMN city_id INT,
    ADD CONSTRAINT fk_rent_ad_city_id FOREIGN KEY (city_id) REFERENCES cities (id) ON DELETE CASCADE;

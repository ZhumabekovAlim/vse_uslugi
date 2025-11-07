ALTER TABLE intercity_orders
    ADD COLUMN driver_id INT NULL AFTER passenger_id,
    ADD COLUMN creator_role ENUM('passenger','driver') NOT NULL DEFAULT 'passenger' AFTER driver_id;

ALTER TABLE intercity_orders
    MODIFY passenger_id INT NULL;

ALTER TABLE intercity_orders
    ADD CONSTRAINT fk_intercity_orders_driver FOREIGN KEY (driver_id) REFERENCES drivers(id) ON UPDATE CASCADE ON DELETE SET NULL;

UPDATE intercity_orders SET creator_role = 'passenger' WHERE creator_role IS NULL;

ALTER TABLE intercity_orders
    DROP FOREIGN KEY fk_intercity_orders_driver;

ALTER TABLE intercity_orders
    DROP COLUMN creator_role,
    DROP COLUMN driver_id,
    MODIFY passenger_id INT NOT NULL;

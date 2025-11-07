ALTER TABLE intercity_orders
    ADD COLUMN contact_phone VARCHAR(32) NOT NULL DEFAULT '' AFTER price;

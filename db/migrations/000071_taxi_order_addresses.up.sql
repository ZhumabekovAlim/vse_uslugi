CREATE TABLE order_addresses
(
    id        INT AUTO_INCREMENT PRIMARY KEY,
    order_id  INT      NOT NULL,
    seq       INT      NOT NULL,
    lon       DOUBLE   NOT NULL,
    lat       DOUBLE   NOT NULL,
    address   VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_order_addresses_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);
CREATE INDEX idx_order_addresses_order_seq ON order_addresses (order_id, seq);

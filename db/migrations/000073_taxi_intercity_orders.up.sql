CREATE TABLE taxi_intercity_orders
(
    id             INT AUTO_INCREMENT PRIMARY KEY,
    client_id      INT                              NOT NULL,
    from_city      VARCHAR(255)                     NOT NULL,
    to_city        VARCHAR(255)                     NOT NULL,
    trip_type      ENUM('with_companions', 'parcel', 'without_companions') NOT NULL,
    comment        TEXT,
    price          DECIMAL(10, 2)                   NOT NULL,
    departure_date DATE                             NOT NULL,
    status         ENUM('open', 'closed')           NOT NULL DEFAULT 'open',
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    closed_at      TIMESTAMP NULL,
    CONSTRAINT fk_taxi_intercity_orders_client
        FOREIGN KEY (client_id) REFERENCES users (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    INDEX idx_taxi_intercity_orders_status (status),
    INDEX idx_taxi_intercity_orders_departure_date (departure_date),
    INDEX idx_taxi_intercity_orders_from_to (from_city, to_city)
);

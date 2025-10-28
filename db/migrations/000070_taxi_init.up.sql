CREATE TABLE drivers
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    user_id     INT NOT NULL,
    status      ENUM('offline', 'free', 'busy') NOT NULL DEFAULT 'offline',
    car_model   VARCHAR(64),
    car_color   VARCHAR(32),
    plate       VARCHAR(32),
    rating      DECIMAL(3, 2) DEFAULT 5.00,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_drivers_user
        FOREIGN KEY (user_id) REFERENCES users (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    CONSTRAINT ux_drivers_user UNIQUE (user_id)
);

CREATE TABLE orders
(
    id                INT AUTO_INCREMENT PRIMARY KEY,
    passenger_id      INT            NOT NULL,
    driver_id         INT,
    from_lon          DOUBLE         NOT NULL,
    from_lat          DOUBLE         NOT NULL,
    to_lon            DOUBLE         NOT NULL,
    to_lat            DOUBLE         NOT NULL,
    distance_m        INT            NOT NULL,
    eta_s             INT            NOT NULL,
    recommended_price INT            NOT NULL,
    client_price      INT            NOT NULL,
    payment_method    ENUM('online', 'cash') NOT NULL,
    status            ENUM('created', 'searching', 'accepted', 'arrived', 'picked_up', 'completed', 'paid', 'closed', 'not_found', 'canceled') NOT NULL DEFAULT 'created',
    notes             VARCHAR(255),
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_orders_passenger
        FOREIGN KEY (passenger_id) REFERENCES users (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    CONSTRAINT fk_orders_driver
        FOREIGN KEY (driver_id) REFERENCES drivers (id)
            ON UPDATE CASCADE
            ON DELETE SET NULL
);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_created ON orders (created_at);

CREATE TABLE order_dispatch
(
    id           INT AUTO_INCREMENT PRIMARY KEY,
    order_id     INT NOT NULL,
    radius_m     INT NOT NULL,
    next_tick_at TIMESTAMP NOT NULL,
    state        ENUM('searching', 'assigned', 'finished') NOT NULL DEFAULT 'searching',
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_dispatch_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);
CREATE UNIQUE INDEX ux_order_dispatch_order ON order_dispatch (order_id);

CREATE TABLE driver_order_offers
(
    id         INT AUTO_INCREMENT PRIMARY KEY,
    order_id   INT      NOT NULL,
    driver_id  INT      NOT NULL,
    state      ENUM('pending', 'accepted', 'declined', 'expired', 'closed') NOT NULL DEFAULT 'pending',
    ttl_at     TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_offer_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    CONSTRAINT fk_offer_driver
        FOREIGN KEY (driver_id) REFERENCES drivers (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);
CREATE INDEX idx_offers_order ON driver_order_offers (order_id);
CREATE UNIQUE INDEX ux_offer_unique ON driver_order_offers (order_id, driver_id);

CREATE TABLE payments
(
    id              INT AUTO_INCREMENT PRIMARY KEY,
    order_id        INT NOT NULL,
    amount          INT NOT NULL,
    provider        ENUM('airbapay') NOT NULL,
    state           ENUM('created', 'authorized', 'paid', 'failed') NOT NULL DEFAULT 'created',
    provider_txn_id VARCHAR(64),
    payload_json    JSON,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_pay_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);

CREATE TABLE payment_webhooks
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    provider    ENUM('airbapay') NOT NULL,
    signature   VARCHAR(256) NOT NULL,
    body_json   JSON          NOT NULL,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_price_history
(
    id         INT AUTO_INCREMENT PRIMARY KEY,
    order_id   INT NOT NULL,
    old_price  INT NOT NULL,
    new_price  INT NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_price_history_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);
CREATE INDEX idx_price_history_order ON order_price_history (order_id);

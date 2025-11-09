CREATE TABLE driver_reviews
(
    id         INT AUTO_INCREMENT PRIMARY KEY,
    driver_id  INT NOT NULL,
    order_id   INT NOT NULL,
    rating     DECIMAL(3, 2),
    comment    TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_driver_reviews_driver
        FOREIGN KEY (driver_id) REFERENCES drivers (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    CONSTRAINT fk_driver_reviews_order
        FOREIGN KEY (order_id) REFERENCES orders (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,
    CONSTRAINT ux_driver_reviews_order UNIQUE (order_id)
);

CREATE INDEX idx_driver_reviews_driver ON driver_reviews (driver_id);

CREATE TABLE courier_reviews (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    courier_id BIGINT UNSIGNED NOT NULL,
    order_id BIGINT UNSIGNED NOT NULL,
    rating DECIMAL(3, 2) NULL,
    comment TEXT NULL,
    courier_rating DECIMAL(3, 2) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_courier_reviews_courier FOREIGN KEY (courier_id) REFERENCES couriers(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT fk_courier_reviews_order FOREIGN KEY (order_id) REFERENCES courier_orders(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT ux_courier_reviews_order UNIQUE (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_courier_reviews_courier ON courier_reviews (courier_id);

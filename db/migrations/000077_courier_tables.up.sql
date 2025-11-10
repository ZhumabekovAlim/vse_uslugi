CREATE TABLE couriers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100) NULL,
    courier_photo VARCHAR(255) NOT NULL,
    iin VARCHAR(32) NOT NULL,
    date_of_birth DATE NOT NULL,
    id_card_front VARCHAR(255) NOT NULL,
    id_card_back VARCHAR(255) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    rating DECIMAL(3,2) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_couriers_user (user_id),
    UNIQUE KEY uq_couriers_iin (iin),
    CONSTRAINT fk_couriers_users FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE courier_orders (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    sender_id BIGINT UNSIGNED NOT NULL,
    courier_id BIGINT UNSIGNED NULL,
    distance_m INT NOT NULL,
    eta_seconds INT NOT NULL,
    recommended_price INT NOT NULL,
    client_price INT NOT NULL,
    payment_method VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_courier_orders_sender FOREIGN KEY (sender_id) REFERENCES users(id),
    CONSTRAINT fk_courier_orders_courier FOREIGN KEY (courier_id) REFERENCES couriers(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_courier_orders_sender ON courier_orders(sender_id, created_at);
CREATE INDEX idx_courier_orders_courier ON courier_orders(courier_id, created_at);

CREATE TABLE courier_order_points (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    seq INT NOT NULL,
    address VARCHAR(255) NOT NULL,
    lat DOUBLE NOT NULL,
    lon DOUBLE NOT NULL,
    entrance VARCHAR(64) NULL,
    apt VARCHAR(64) NULL,
    floor VARCHAR(64) NULL,
    intercom VARCHAR(64) NULL,
    phone VARCHAR(32) NULL,
    comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_courier_points_order FOREIGN KEY (order_id) REFERENCES courier_orders(id) ON DELETE CASCADE,
    UNIQUE KEY uq_courier_order_point (order_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE courier_offers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    courier_id BIGINT UNSIGNED NOT NULL,
    price INT NOT NULL,
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_courier_offer (order_id, courier_id),
    CONSTRAINT fk_courier_offers_order FOREIGN KEY (order_id) REFERENCES courier_orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_courier_offers_courier FOREIGN KEY (courier_id) REFERENCES couriers(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE courier_order_status_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL,
    note TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_courier_status_order FOREIGN KEY (order_id) REFERENCES courier_orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_courier_status_history_order ON courier_order_status_history(order_id, created_at);

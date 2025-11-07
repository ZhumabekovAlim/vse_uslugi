-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  phone VARCHAR(32) NOT NULL UNIQUE,
  role ENUM('passenger','driver','admin') NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS drivers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  status ENUM('offline','free','busy') NOT NULL DEFAULT 'offline',
  car_model VARCHAR(64),
  car_color VARCHAR(32),
  car_number VARCHAR(32),
  tech_passport VARCHAR(128) NOT NULL,
  car_photo_front VARCHAR(255) NOT NULL,
  car_photo_back VARCHAR(255) NOT NULL,
  car_photo_left VARCHAR(255) NOT NULL,
  car_photo_right VARCHAR(255) NOT NULL,
  driver_photo VARCHAR(255) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  iin VARCHAR(32) NOT NULL,
  id_card_front VARCHAR(255) NOT NULL,
  id_card_back VARCHAR(255) NOT NULL,
  rating DECIMAL(3,2) DEFAULT 5.00,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_drivers_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  passenger_id BIGINT NOT NULL,
  driver_id BIGINT NULL,
  from_lon DOUBLE NOT NULL,
  from_lat DOUBLE NOT NULL,
  to_lon DOUBLE NOT NULL,
  to_lat DOUBLE NOT NULL,
  distance_m INT NOT NULL,
  eta_s INT NOT NULL,
  recommended_price INT NOT NULL,
  client_price INT NOT NULL,
  payment_method ENUM('online','cash') NOT NULL,
  status ENUM('created','searching','accepted','arrived','picked_up','completed','paid','closed','not_found','canceled') NOT NULL DEFAULT 'created',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  notes VARCHAR(255),
  CONSTRAINT fk_orders_passenger FOREIGN KEY (passenger_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
  CONSTRAINT fk_orders_driver FOREIGN KEY (driver_id) REFERENCES drivers(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created ON orders(created_at);

CREATE TABLE IF NOT EXISTS order_dispatch (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id BIGINT NOT NULL,
  radius_m INT NOT NULL,
  next_tick_at TIMESTAMP NOT NULL,
  state ENUM('searching','assigned','finished') NOT NULL DEFAULT 'searching',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_dispatch_order FOREIGN KEY (order_id) REFERENCES orders(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE UNIQUE INDEX ux_order_dispatch_order ON order_dispatch(order_id);

CREATE TABLE IF NOT EXISTS driver_order_offers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id BIGINT NOT NULL,
  driver_id BIGINT NOT NULL,
  state ENUM('pending','accepted','declined','expired','closed') NOT NULL DEFAULT 'pending',
  ttl_at TIMESTAMP NOT NULL,
  driver_price INT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_offer_order FOREIGN KEY (order_id) REFERENCES orders(id) ON UPDATE CASCADE ON DELETE RESTRICT,
  CONSTRAINT fk_offer_driver FOREIGN KEY (driver_id) REFERENCES drivers(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_offers_order ON driver_order_offers(order_id);
CREATE UNIQUE INDEX ux_offer_unique ON driver_order_offers(order_id, driver_id);

CREATE TABLE IF NOT EXISTS payments (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id BIGINT NOT NULL,
  amount INT NOT NULL,
  provider ENUM('airbapay') NOT NULL,
  state ENUM('created','authorized','paid','failed') NOT NULL DEFAULT 'created',
  provider_txn_id VARCHAR(64),
  payload_json JSON,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_pay_order FOREIGN KEY (order_id) REFERENCES orders(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS payment_webhooks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  provider ENUM('airbapay') NOT NULL,
  signature VARCHAR(256) NOT NULL,
  body_json JSON NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS order_price_history (
  order_id BIGINT NOT NULL,
  old_price INT NOT NULL,
  new_price INT NOT NULL,
  changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_price_history_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +migrate Down
DROP TABLE IF EXISTS order_price_history;
DROP TABLE IF EXISTS payment_webhooks;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS driver_order_offers;
DROP TABLE IF EXISTS order_dispatch;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS drivers;
DROP TABLE IF EXISTS users;

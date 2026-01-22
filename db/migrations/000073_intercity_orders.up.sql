CREATE TABLE IF NOT EXISTS intercity_orders (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  passenger_id BIGINT NOT NULL,
  from_location VARCHAR(255) NOT NULL,
  to_location VARCHAR(255) NOT NULL,
  trip_type ENUM('companions','parcel','solo') NOT NULL,
  comment TEXT,
  price BIGINT NOT NULL,
  contact_phone VARCHAR(32) NOT NULL,
  departure_date DATE NOT NULL,
  departure_time TIME NULL,
  status ENUM('open','closed') NOT NULL DEFAULT 'open',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  closed_at TIMESTAMP NULL,
  CONSTRAINT fk_intercity_orders_passenger FOREIGN KEY (passenger_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_intercity_orders_status ON intercity_orders(status);
CREATE INDEX idx_intercity_orders_departure_date ON intercity_orders(departure_date);
CREATE INDEX idx_intercity_orders_departure_time ON intercity_orders(departure_time);

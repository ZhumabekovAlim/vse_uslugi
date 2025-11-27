CREATE TABLE IF NOT EXISTS business_accounts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    business_user_id BIGINT NOT NULL,
    seats_total INT NOT NULL DEFAULT 0,
    seats_used INT NOT NULL DEFAULT 0,
    status ENUM('active', 'suspended') NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_business_accounts_user (business_user_id),
    CONSTRAINT fk_business_accounts_user FOREIGN KEY (business_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS business_workers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    business_user_id BIGINT NOT NULL,
    worker_user_id BIGINT NOT NULL,
    login VARCHAR(255) NOT NULL,
    chat_id BIGINT NOT NULL,
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_business_workers_login (login),
    UNIQUE KEY uniq_business_workers_worker (worker_user_id),
    INDEX idx_business_workers_business_user_id (business_user_id),
    INDEX idx_business_workers_chat_id (chat_id),
    CONSTRAINT fk_business_workers_business_user FOREIGN KEY (business_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_business_workers_worker_user FOREIGN KEY (worker_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_business_workers_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS business_worker_listings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    business_user_id BIGINT NOT NULL,
    worker_user_id BIGINT NOT NULL,
    listing_type ENUM('ad', 'service', 'work', 'work_ad', 'rent', 'rent_ad') NOT NULL,
    listing_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_worker_listing (listing_type, listing_id),
    INDEX idx_business_worker_listings_business (business_user_id),
    INDEX idx_business_worker_listings_worker (worker_user_id),
    CONSTRAINT fk_business_worker_listings_business_user FOREIGN KEY (business_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_business_worker_listings_worker_user FOREIGN KEY (worker_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS business_seat_purchases (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    business_user_id BIGINT NOT NULL,
    seats INT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    provider VARCHAR(255) DEFAULT NULL,
    state VARCHAR(255) DEFAULT NULL,
    provider_txn_id VARCHAR(255) DEFAULT NULL,
    payload_json JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_business_seat_purchases_user (business_user_id),
    CONSTRAINT fk_business_seat_purchases_user FOREIGN KEY (business_user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

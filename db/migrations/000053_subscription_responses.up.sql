CREATE TABLE IF NOT EXISTS subscription_responses (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    packs INT NOT NULL,
    status VARCHAR(20) NOT NULL,
    renews_at TIMESTAMP NOT NULL,
    monthly_quota INT NOT NULL,
    remaining INT NOT NULL,
    provider_subscription_id VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL,
    INDEX idx_subscription_responses_user (user_id)
);

use naimudb;
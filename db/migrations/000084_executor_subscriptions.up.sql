CREATE TABLE IF NOT EXISTS executor_subscriptions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_type ENUM('service', 'rent', 'work') NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL,
    UNIQUE KEY uniq_executor_subscription_user_type (user_id, subscription_type),
    INDEX idx_executor_subscriptions_user (user_id),
    INDEX idx_executor_subscriptions_expires_at (expires_at)
);

DROP TABLE IF EXISTS subscription_slots;

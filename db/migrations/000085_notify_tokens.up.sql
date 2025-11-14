CREATE TABLE IF NOT EXISTS notify_tokens (
                                             id INT AUTO_INCREMENT PRIMARY KEY,
                                             user_id INT NOT NULL,
                                             token VARCHAR(255) NOT NULL,
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                             updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Индекс для ускорения поиска по user_id
CREATE INDEX idx_notify_tokens_user ON notify_tokens (user_id);

-- Индекс для ускорения поиска по token
CREATE UNIQUE INDEX ux_notify_tokens_token ON notify_tokens (token);
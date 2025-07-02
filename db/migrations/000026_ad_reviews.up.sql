CREATE TABLE ad_reviews
(
    id         INT AUTO_INCREMENT PRIMARY KEY,
    user_id    INT           NOT NULL,
    ad_id    INT           NOT NULL,
    rating     DECIMAL(3, 2) NOT NULL,
    review     TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (ad_id) REFERENCES ad (id) ON DELETE CASCADE
);
use naimudb;
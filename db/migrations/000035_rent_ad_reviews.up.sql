CREATE TABLE rent_ad_reviews
(
    id         INT AUTO_INCREMENT PRIMARY KEY,
    user_id    INT           NOT NULL,
    rent_ad_id    INT           NOT NULL,
    rating     DECIMAL(3, 2) NOT NULL,
    review     TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (rent_ad_id) REFERENCES rent_ad (id) ON DELETE CASCADE
);
use naimudb;
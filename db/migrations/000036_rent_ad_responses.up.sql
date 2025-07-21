CREATE TABLE rent_ad_responses
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    user_id     INT          NOT NULL,
    rent_ad_id     INT          NOT NULL,
    price       DECIMAL(10, 2)        not null,
    description TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (rent_ad_id) REFERENCES rent_ad (id) ON DELETE CASCADE
);

use naimudb;
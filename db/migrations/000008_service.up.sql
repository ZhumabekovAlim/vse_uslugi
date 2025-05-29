CREATE TABLE service
(
    id            INT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255)   NOT NULL,
    address       VARCHAR(255)   NOT NULL,
    price         DECIMAL(10, 2) NOT NULL,
    user_id       INT            NOT NULL,
    images        TEXT,
    category_id   INT            NOT NULL,
    description   TEXT,
    avg_rating    DECIMAL(3, 2),
    liked boolean,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

use naimudb;
CREATE TABLE users
(
    id            INT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    surname       VARCHAR(255) NOT NULL,
    middlename    VARCHAR(255),
    phone         VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password      VARCHAR(255) NOT NULL,
    city_id       INT,
    years_of_exp  INT,
    doc_of_proof  VARCHAR(255),
    review_rating DECIMAL(10, 2),
    role          VARCHAR(255) NOT NULL,
    latitude      VARCHAR(255),
    longitude     VARCHAR(255),
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    refresh_token VARCHAR(255),
    expires_at DATETIME,
    FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE CASCADE
);
USE naimudb;
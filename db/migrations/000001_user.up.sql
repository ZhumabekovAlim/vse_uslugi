CREATE TABLE users
(
    id            INT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255)   NOT NULL,
    phone         VARCHAR(255)   NOT NULL,
    email         VARCHAR(255)   NOT NULL,
    password      VARCHAR(255)   NOT NULL,
    city          VARCHAR(255)   NOT NULL,
    years_of_exp  INT,
    doc_of_proof  VARCHAR(255),
    review_rating DECIMAL(10, 2) NOT NULL,
    role          VARCHAR(255)   NOT NULL,
    latitude      VARCHAR(255)   NOT NULL,
    longitude     VARCHAR(255)   NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
USE naimudb;
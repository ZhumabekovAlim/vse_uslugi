CREATE TABLE work_ad
(
    id              INT AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(255)   NOT NULL,
    address         VARCHAR(255)   NOT NULL,
    price           BIGINT         NOT NULL,
    user_id         INT            NOT NULL,
    images          TEXT,
    category_id     INT            NOT NULL,
    subcategory_id  INT,
    description     TEXT,
    avg_rating      DECIMAL(3, 2),
    top             VARCHAR(255),
    liked           boolean,
    status          varchar(255),
    work_experience varchar(255),
    city_id         INT            NOT NULL,
    schedule        varchar(255),
    distance_work   varchar(255),
    payment_period  varchar(255),
    latitude        varchar(255),
    longitude       varchar(255),
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE,
    FOREIGN KEY (city_id) REFERENCES cities (id) ON DELETE CASCADE
);

use naimudb;

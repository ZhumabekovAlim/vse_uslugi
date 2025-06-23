CREATE TABLE complaints
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    service_id  INT  NOT NULL,
    user_id     INT  NOT NULL,
    description TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES service (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
use naimudb;
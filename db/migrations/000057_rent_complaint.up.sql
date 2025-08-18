CREATE TABLE rent_complaints
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    rent_id     INT  NOT NULL,
    user_id     INT  NOT NULL,
    description TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rent_id) REFERENCES rent (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
use naimudb;

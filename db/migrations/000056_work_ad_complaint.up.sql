CREATE TABLE work_ad_complaints
(
    id          INT AUTO_INCREMENT PRIMARY KEY,
    work_ad_id  INT  NOT NULL,
    user_id     INT  NOT NULL,
    description TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_ad_id) REFERENCES work_ad (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
use naimudb;

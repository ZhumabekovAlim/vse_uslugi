CREATE TABLE IF NOT EXISTS user_categories (
                                               user_id INT,
                                               category_id INT,
                                               PRIMARY KEY (user_id, category_id),
                                               FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                                               FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);
use naimudb;
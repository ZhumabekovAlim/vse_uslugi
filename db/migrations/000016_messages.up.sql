CREATE TABLE IF NOT EXISTS messages (
                                        id INT AUTO_INCREMENT PRIMARY KEY,
                                        sender_id INT NOT NULL,
                                        receiver_id INT NOT NULL,
                                        text TEXT NOT NULL,
                                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                        chat_id INT NOT NULL,

                                        FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
                                        FOREIGN KEY (receiver_id) REFERENCES users(id) ON DELETE CASCADE,
                                        FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
);
use naimudb;
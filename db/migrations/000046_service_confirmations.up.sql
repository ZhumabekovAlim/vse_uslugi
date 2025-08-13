CREATE TABLE service_confirmations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    service_id INT NOT NULL,
    chat_id INT NOT NULL,
    client_id INT NOT NULL,
    performer_id INT NOT NULL,
    confirmed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (performer_id) REFERENCES users(id) ON DELETE CASCADE
);

use naimudb;

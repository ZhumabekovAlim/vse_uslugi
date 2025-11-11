CREATE TABLE invoice_targets (
    id INT AUTO_INCREMENT PRIMARY KEY,
    invoice_id INT NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id BIGINT NOT NULL,
    payload_json JSON NULL,
    processed_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_invoice_targets_invoice FOREIGN KEY (invoice_id) REFERENCES invoices (inv_id) ON DELETE CASCADE,
    UNIQUE KEY ux_invoice_target (invoice_id, target_type, target_id)
);

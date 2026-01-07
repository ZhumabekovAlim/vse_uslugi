ALTER TABLE business_accounts
    ADD COLUMN seats_expires_at DATETIME NULL DEFAULT NULL AFTER status;

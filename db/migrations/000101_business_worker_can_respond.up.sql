ALTER TABLE business_workers
    ADD COLUMN can_respond TINYINT(1) NOT NULL DEFAULT 0 AFTER status;

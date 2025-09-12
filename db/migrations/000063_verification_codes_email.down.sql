ALTER TABLE verification_codes
    DROP COLUMN email,
    MODIFY phone VARCHAR(20) NOT NULL;

ALTER TABLE drivers
    CHANGE plate car_number VARCHAR(32),
    ADD COLUMN tech_passport VARCHAR(128) NOT NULL AFTER car_number,
    ADD COLUMN car_photo_front VARCHAR(255) NOT NULL AFTER tech_passport,
    ADD COLUMN car_photo_back VARCHAR(255) NOT NULL AFTER car_photo_front,
    ADD COLUMN car_photo_left VARCHAR(255) NOT NULL AFTER car_photo_back,
    ADD COLUMN car_photo_right VARCHAR(255) NOT NULL AFTER car_photo_left,
    ADD COLUMN driver_photo VARCHAR(255) NOT NULL AFTER car_photo_right,
    ADD COLUMN phone VARCHAR(32) NOT NULL AFTER driver_photo,
    ADD COLUMN iin VARCHAR(32) NOT NULL AFTER phone,
    ADD COLUMN id_card_front VARCHAR(255) NOT NULL AFTER iin,
    ADD COLUMN id_card_back VARCHAR(255) NOT NULL AFTER id_card_front;

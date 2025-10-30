ALTER TABLE drivers
    CHANGE car_number plate VARCHAR(32),
    DROP COLUMN tech_passport,
    DROP COLUMN car_photo_front,
    DROP COLUMN car_photo_back,
    DROP COLUMN car_photo_left,
    DROP COLUMN car_photo_right,
    DROP COLUMN driver_photo,
    DROP COLUMN phone,
    DROP COLUMN iin,
    DROP COLUMN id_card_front,
    DROP COLUMN id_card_back;

ALTER TABLE reviews
    ADD CONSTRAINT ux_reviews_user_service UNIQUE (user_id, service_id);

ALTER TABLE ad_reviews
    ADD CONSTRAINT ux_ad_reviews_user_ad UNIQUE (user_id, ad_id);

ALTER TABLE rent_reviews
    ADD CONSTRAINT ux_rent_reviews_user_rent UNIQUE (user_id, rent_id);

ALTER TABLE rent_ad_reviews
    ADD CONSTRAINT ux_rent_ad_reviews_user_rent_ad UNIQUE (user_id, rent_ad_id);

ALTER TABLE work_reviews
    ADD CONSTRAINT ux_work_reviews_user_work UNIQUE (user_id, work_id);

ALTER TABLE work_ad_reviews
    ADD CONSTRAINT ux_work_ad_reviews_user_work_ad UNIQUE (user_id, work_ad_id);

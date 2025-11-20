ALTER TABLE reviews
    DROP INDEX ux_reviews_user_service;

ALTER TABLE ad_reviews
    DROP INDEX ux_ad_reviews_user_ad;

ALTER TABLE rent_reviews
    DROP INDEX ux_rent_reviews_user_rent;

ALTER TABLE rent_ad_reviews
    DROP INDEX ux_rent_ad_reviews_user_rent_ad;

ALTER TABLE work_reviews
    DROP INDEX ux_work_reviews_user_work;

ALTER TABLE work_ad_reviews
    DROP INDEX ux_work_ad_reviews_user_work_ad;

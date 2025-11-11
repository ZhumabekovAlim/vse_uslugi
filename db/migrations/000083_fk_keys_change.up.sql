ALTER TABLE `ad`                       DROP FOREIGN KEY `ad_ibfk_1`,
                                       ADD CONSTRAINT `fk_ad_user_id`                       FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_complaints`            DROP FOREIGN KEY `ad_complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_ad_complaints_user_id`            FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_confirmations`         DROP FOREIGN KEY `ad_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_ad_confirmations_client_id`       FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_confirmations`         DROP FOREIGN KEY `ad_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_ad_confirmations_performer_id`    FOREIGN KEY (`performer_id`)         REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_favorites`             DROP FOREIGN KEY `ad_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_ad_favorites_user_id`             FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_responses`             DROP FOREIGN KEY `ad_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_ad_responses_user_id`             FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `ad_reviews`               DROP FOREIGN KEY `ad_reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_ad_reviews_user_id`               FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `chats`                    DROP FOREIGN KEY `chats_ibfk_1`,
                                       ADD CONSTRAINT `fk_chats_user1_id`                   FOREIGN KEY (`user1_id`)             REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `chats`                    DROP FOREIGN KEY `chats_ibfk_2`,
                                       ADD CONSTRAINT `fk_chats_user2_id`                   FOREIGN KEY (`user2_id`)             REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `complaints`               DROP FOREIGN KEY `complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_complaints_user_id`               FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `couriers`                 DROP FOREIGN KEY `fk_couriers_users`,
                                       ADD CONSTRAINT `fk_couriers_user_id`                 FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `courier_orders`           DROP FOREIGN KEY `fk_courier_orders_sender`,
                                       ADD CONSTRAINT `fk_courier_orders_sender_id`         FOREIGN KEY (`sender_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `drivers`                  DROP FOREIGN KEY `fk_drivers_user`,
                                       ADD CONSTRAINT `fk_drivers_user_id`                  FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `intercity_orders`         DROP FOREIGN KEY `fk_intercity_orders_passenger`,
                                       ADD CONSTRAINT `fk_intercity_orders_passenger_id`    FOREIGN KEY (`passenger_id`)         REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `invoices`                 DROP FOREIGN KEY `invoices_ibfk_1`,
                                       ADD CONSTRAINT `fk_invoices_user_id`                 FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `messages`                 DROP FOREIGN KEY `messages_ibfk_1`,
                                       ADD CONSTRAINT `fk_messages_sender_id`               FOREIGN KEY (`sender_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `messages`                 DROP FOREIGN KEY `messages_ibfk_2`,
                                       ADD CONSTRAINT `fk_messages_receiver_id`             FOREIGN KEY (`receiver_id`)          REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `orders`                   DROP FOREIGN KEY `fk_orders_passenger`,
                                       ADD CONSTRAINT `fk_orders_passenger_id`              FOREIGN KEY (`passenger_id`)         REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent`                     DROP FOREIGN KEY `rent_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_user_id`                     FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad`                  DROP FOREIGN KEY `rent_ad_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_ad_user_id`                  FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_complaints`       DROP FOREIGN KEY `rent_ad_complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_rent_ad_complaints_user_id`       FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_confirmations`    DROP FOREIGN KEY `rent_ad_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_rent_ad_confirmations_client_id`  FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_confirmations`    DROP FOREIGN KEY `rent_ad_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_rent_ad_confirmations_performer_id` FOREIGN KEY (`performer_id`)      REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_favorites`        DROP FOREIGN KEY `rent_ad_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_ad_favorites_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_responses`        DROP FOREIGN KEY `rent_ad_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_ad_responses_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_ad_reviews`          DROP FOREIGN KEY `rent_ad_reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_ad_reviews_user_id`          FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_complaints`          DROP FOREIGN KEY `rent_complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_rent_complaints_user_id`          FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_confirmations`       DROP FOREIGN KEY `rent_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_rent_confirmations_client_id`     FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_confirmations`       DROP FOREIGN KEY `rent_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_rent_confirmations_performer_id`  FOREIGN KEY (`performer_id`)         REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_favorites`           DROP FOREIGN KEY `rent_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_favorites_user_id`           FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_responses`           DROP FOREIGN KEY `rent_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_responses_user_id`           FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `rent_reviews`             DROP FOREIGN KEY `rent_reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_rent_reviews_user_id`             FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `reviews`                  DROP FOREIGN KEY `reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_reviews_user_id`                  FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `service`                  DROP FOREIGN KEY `service_ibfk_1`,
                                       ADD CONSTRAINT `fk_service_user_id`                  FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `service_confirmations`    DROP FOREIGN KEY `service_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_service_confirmations_client_id`  FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `service_confirmations`    DROP FOREIGN KEY `service_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_service_confirmations_performer_id` FOREIGN KEY (`performer_id`)       REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `service_favorites`        DROP FOREIGN KEY `service_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_service_favorites_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `service_responses`        DROP FOREIGN KEY `service_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_service_responses_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `user_categories`          DROP FOREIGN KEY `user_categories_ibfk_1`,
                                       ADD CONSTRAINT `fk_user_categories_user_id`          FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work`                     DROP FOREIGN KEY `work_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_user_id`                     FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad`                  DROP FOREIGN KEY `work_ad_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_ad_user_id`                  FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_complaints`       DROP FOREIGN KEY `work_ad_complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_work_ad_complaints_user_id`       FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_confirmations`    DROP FOREIGN KEY `work_ad_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_work_ad_confirmations_client_id`  FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_confirmations`    DROP FOREIGN KEY `work_ad_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_work_ad_confirmations_performer_id` FOREIGN KEY (`performer_id`)       REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_favorites`        DROP FOREIGN KEY `work_ad_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_ad_favorites_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_responses`        DROP FOREIGN KEY `work_ad_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_ad_responses_user_id`        FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_ad_reviews`          DROP FOREIGN KEY `work_ad_reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_ad_reviews_user_id`          FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_complaints`          DROP FOREIGN KEY `work_complaints_ibfk_2`,
                                       ADD CONSTRAINT `fk_work_complaints_user_id`          FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_confirmations`       DROP FOREIGN KEY `work_confirmations_ibfk_3`,
                                       ADD CONSTRAINT `fk_work_confirmations_client_id`     FOREIGN KEY (`client_id`)            REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_confirmations`       DROP FOREIGN KEY `work_confirmations_ibfk_4`,
                                       ADD CONSTRAINT `fk_work_confirmations_performer_id`  FOREIGN KEY (`performer_id`)         REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_favorites`           DROP FOREIGN KEY `work_favorites_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_favorites_user_id`           FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_responses`           DROP FOREIGN KEY `work_responses_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_responses_user_id`           FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `work_reviews`             DROP FOREIGN KEY `work_reviews_ibfk_1`,
                                       ADD CONSTRAINT `fk_work_reviews_user_id`             FOREIGN KEY (`user_id`)              REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

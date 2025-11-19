ALTER TABLE categories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;
ALTER TABLE rent_categories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;
ALTER TABLE work_categories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;

use naimudb;

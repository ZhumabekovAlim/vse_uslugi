ALTER TABLE subcategories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;
ALTER TABLE rent_subcategories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;
ALTER TABLE work_subcategories ADD COLUMN name_kz VARCHAR(255) DEFAULT '' AFTER name;

use naimudb;

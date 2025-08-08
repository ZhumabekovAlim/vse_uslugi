ALTER TABLE cities ADD COLUMN parent_id INT NULL;
ALTER TABLE cities ADD CONSTRAINT fk_cities_parent FOREIGN KEY (parent_id) REFERENCES cities(id) ON DELETE SET NULL;

use naimudb;

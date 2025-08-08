ALTER TABLE cities DROP FOREIGN KEY fk_cities_parent;
ALTER TABLE cities DROP COLUMN parent_id;

use naimudb;

CREATE TABLE category_subcategory (
                                      category_id INT NOT NULL,
                                      subcategory_id INT NOT NULL,
                                      PRIMARY KEY (category_id, subcategory_id),
                                      FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
                                      FOREIGN KEY (subcategory_id) REFERENCES subcategories(id) ON DELETE CASCADE
);
use naimudb;
ALTER TABLE authors ADD CONSTRAINT unique_author UNIQUE (name, last_name);
ALTER TABLE publishers ADD CONSTRAINT unique_publisher_name UNIQUE (name);

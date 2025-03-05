-- Add the slug column to the books table
ALTER TABLE books
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the book title and id
UPDATE books
SET slug = LOWER(REGEXP_REPLACE(short_title, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || id;

-- Add a trigger to automatically update the slug when a new book is inserted or updated
CREATE OR REPLACE FUNCTION update_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(REGEXP_REPLACE(NEW.short_title, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || NEW.id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER books_slug_trigger
BEFORE INSERT OR UPDATE ON books
FOR EACH ROW
EXECUTE FUNCTION update_slug();


-- PUBLISHER SLUGS

ALTER TABLE publishers
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the book title and id
UPDATE publishers
SET slug = LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || id;

-- Add a trigger to automatically update the slug when a new book is inserted or updated
CREATE OR REPLACE FUNCTION update_publisher_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(REGEXP_REPLACE(NEW.name, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || NEW.id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER publishers_slug_trigger
BEFORE INSERT OR UPDATE ON publishers
FOR EACH ROW
EXECUTE FUNCTION update_publisher_slug();

-- AUTHOR SLUGS


-- Add the slug column to the authors table
ALTER TABLE authors
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the author's last name and first name
UPDATE authors
SET slug = LOWER(REGEXP_REPLACE(last_name, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9]+', '-', 'g'));

-- Add a trigger to automatically update the slug when a new author is inserted or updated
CREATE OR REPLACE FUNCTION update_author_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(REGEXP_REPLACE(NEW.last_name, '[^a-zA-Z0-9]+', '-', 'g')) || '-' || LOWER(REGEXP_REPLACE(NEW.name, '[^a-zA-Z0-9]+', '-', 'g'));
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER authors_slug_trigger
BEFORE INSERT OR UPDATE ON authors
FOR EACH ROW
EXECUTE FUNCTION update_author_slug();


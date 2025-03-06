CREATE EXTENSION IF NOT EXISTS unaccent;

-- Add the slug column to the books table
ALTER TABLE books
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the book title and id
UPDATE books
SET slug = LOWER(
  REGEXP_REPLACE(
    UNACCENT(short_title),  -- Normalize accented characters
    '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
  )
) || '-' || id;

-- Add a trigger to automatically update the slug when a new book is inserted or updated
CREATE OR REPLACE FUNCTION update_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(
    REGEXP_REPLACE(
      UNACCENT(NEW.short_title),  -- Normalize accented characters
      '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
    )
  ) || '-' || NEW.id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER books_slug_trigger
BEFORE INSERT OR UPDATE ON books
FOR EACH ROW
EXECUTE FUNCTION update_slug();

-- Add the slug column to the publishers table
ALTER TABLE publishers
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the publisher name and id
UPDATE publishers
SET slug = LOWER(
  REGEXP_REPLACE(
    UNACCENT(name),  -- Normalize accented characters
    '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
  )
) || '-' || id;

-- Add a trigger to automatically update the slug when a new publisher is inserted or updated
CREATE OR REPLACE FUNCTION update_publisher_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(
    REGEXP_REPLACE(
      UNACCENT(NEW.name),  -- Normalize accented characters
      '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
    )
  ) || '-' || NEW.id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER publishers_slug_trigger
BEFORE INSERT OR UPDATE ON publishers
FOR EACH ROW
EXECUTE FUNCTION update_publisher_slug();

-- Add the slug column to the authors table
ALTER TABLE authors
ADD COLUMN slug text UNIQUE;

-- Update the slug column with the author's last name and first name
UPDATE authors
SET slug = LOWER(
  REGEXP_REPLACE(
    UNACCENT(last_name || '-' || name),  -- Normalize accented characters
    '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
  )
);

-- Add a trigger to automatically update the slug when a new author is inserted or updated
CREATE OR REPLACE FUNCTION update_author_slug()
RETURNS TRIGGER AS $$
BEGIN
  NEW.slug := LOWER(
    REGEXP_REPLACE(
      UNACCENT(NEW.last_name || '-' || NEW.name),  -- Normalize accented characters
      '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
    )
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER authors_slug_trigger
BEFORE INSERT OR UPDATE ON authors
FOR EACH ROW
EXECUTE FUNCTION update_author_slug();

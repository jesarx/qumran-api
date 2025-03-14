CREATE EXTENSION IF NOT EXISTS unaccent;

-- Add the slug column to the books table
ALTER TABLE books
ADD COLUMN slug text UNIQUE;

-- Update the slug column with author names and book title
UPDATE books
SET slug = (
  SELECT 
    LOWER(
      REGEXP_REPLACE(
        UNACCENT(COALESCE(a.last_name, '')),
        '[^a-zA-Z0-9]+', '-', 'g'
      ) || '-' ||
      REGEXP_REPLACE(
        UNACCENT(COALESCE(a.name, '')),
        '[^a-zA-Z0-9]+', '-', 'g'
      ) || '-' ||
      REGEXP_REPLACE(
        UNACCENT(b.short_title),
        '[^a-zA-Z0-9]+', '-', 'g'
      )
    )
  FROM books b
  LEFT JOIN authors a ON b.auth_id = a.id
  WHERE b.id = books.id
);

-- Create function to handle duplicate slugs
CREATE OR REPLACE FUNCTION generate_unique_slug(base_slug text) 
RETURNS text AS $$
DECLARE
    unique_slug text := base_slug;
    counter integer := 1;
BEGIN
    LOOP
        -- Check if this slug already exists
        PERFORM FROM books WHERE slug = unique_slug;
        -- If not found, we have a unique slug
        IF NOT FOUND THEN
            RETURN unique_slug;
        END IF;
        -- Otherwise, append a number and try again
        unique_slug := base_slug || '-' || counter;
        counter := counter + 1;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Update slugs to ensure uniqueness
DO $$
DECLARE
    book record;
BEGIN
    FOR book IN SELECT id, slug FROM books
    LOOP
        UPDATE books
        SET slug = generate_unique_slug(slug)
        WHERE id = book.id;
    END LOOP;
END $$;

-- Create function for the trigger
CREATE OR REPLACE FUNCTION update_slug()
RETURNS TRIGGER AS $$
DECLARE
    base_slug text;
BEGIN
    -- Get author information
    SELECT 
        LOWER(
            REGEXP_REPLACE(
                UNACCENT(COALESCE(a.last_name, '')),
                '[^a-zA-Z0-9]+', '-', 'g'
            ) || '-' ||
            REGEXP_REPLACE(
                UNACCENT(COALESCE(a.name, '')),
                '[^a-zA-Z0-9]+', '-', 'g'
            ) || '-' ||
            REGEXP_REPLACE(
                UNACCENT(NEW.short_title),
                '[^a-zA-Z0-9]+', '-', 'g'
            )
        ) INTO base_slug
    FROM authors a
    WHERE a.id = NEW.auth_id;

    -- If no author found, use only the book title
    IF base_slug IS NULL THEN
        base_slug := LOWER(
            REGEXP_REPLACE(
                UNACCENT(NEW.short_title),
                '[^a-zA-Z0-9]+', '-', 'g'
            )
        );
    END IF;

    -- Generate unique slug
    NEW.slug := generate_unique_slug(base_slug);
    
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

-- Create function to handle duplicate slugs
CREATE OR REPLACE FUNCTION generate_unique_publisher_slug(base_slug text) 
RETURNS text AS $$
DECLARE
    unique_slug text := base_slug;
    counter integer := 1;
BEGIN
    LOOP
        -- Check if this slug already exists
        PERFORM FROM publishers WHERE slug = unique_slug;
        -- If not found, we have a unique slug
        IF NOT FOUND THEN
            RETURN unique_slug;
        END IF;
        -- Otherwise, append a number and try again
        unique_slug := base_slug || '-' || counter;
        counter := counter + 1;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Generate base slugs for all publishers
UPDATE publishers
SET slug = LOWER(
  REGEXP_REPLACE(
    UNACCENT(name),  -- Normalize accented characters
    '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
  )
);

-- Update slugs to ensure uniqueness
DO $$
DECLARE
    publisher record;
BEGIN
    FOR publisher IN SELECT id, slug FROM publishers ORDER BY id
    LOOP
        UPDATE publishers
        SET slug = generate_unique_publisher_slug(slug)
        WHERE id = publisher.id;
    END LOOP;
END $$;

-- Create function for the trigger
CREATE OR REPLACE FUNCTION update_publisher_slug()
RETURNS TRIGGER AS $$
DECLARE
    base_slug text;
BEGIN
    base_slug := LOWER(
        REGEXP_REPLACE(
            UNACCENT(NEW.name),  -- Normalize accented characters
            '[^a-zA-Z0-9]+', '-', 'g'  -- Replace non-alphanumeric characters with hyphens
        )
    );
    
    -- Generate unique slug
    NEW.slug := generate_unique_publisher_slug(base_slug);
    
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

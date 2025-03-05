-- Drop the authors_slug_trigger
DROP TRIGGER IF EXISTS authors_slug_trigger ON authors;

-- Drop the update_author_slug function
DROP FUNCTION IF EXISTS update_author_slug;

-- Drop the slug column from the authors table
ALTER TABLE authors
DROP COLUMN IF EXISTS slug;

-- Drop the publishers_slug_trigger
DROP TRIGGER IF EXISTS publishers_slug_trigger ON publishers;

-- Drop the update_publisher_slug function
DROP FUNCTION IF EXISTS update_publisher_slug;

-- Drop the slug column from the publishers table
ALTER TABLE publishers
DROP COLUMN IF EXISTS slug;

-- Drop the books_slug_trigger
DROP TRIGGER IF EXISTS books_slug_trigger ON books;

-- Drop the update_slug function
DROP FUNCTION IF EXISTS update_slug;

-- Drop the slug column from the books table
ALTER TABLE books
DROP COLUMN IF EXISTS slug;


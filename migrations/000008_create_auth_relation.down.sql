ALTER TABLE books
DROP CONSTRAINT fk_books_authors;

ALTER TABLE books
DROP COLUMN auth_id;

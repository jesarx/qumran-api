ALTER TABLE books
ADD COLUMN auth_id bigint NOT NULL;

ALTER TABLE books
ADD CONSTRAINT fk_books_authors
FOREIGN KEY (auth_id)
REFERENCES authors(id);

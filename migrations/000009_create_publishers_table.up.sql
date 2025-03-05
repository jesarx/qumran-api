CREATE TABLE IF NOT EXISTS publishers(
  id bigserial PRIMARY KEY,
  created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  name text NOT NULL,
  version integer NOT NULL DEFAULT 1
);

ALTER TABLE books
ADD COLUMN pub_id bigint NOT NULL;

ALTER TABLE books
ADD CONSTRAINT fk_books_publishers
FOREIGN KEY (pub_id)
REFERENCES publishers(id);

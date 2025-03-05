CREATE INDEX IF NOT EXISTS books_title_idx ON books USING GIN (to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS books_tags_idx ON books USING GIN (tags);

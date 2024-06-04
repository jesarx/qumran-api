ALTER TABLE books ADD CONSTRAINT tags_lenght_check CHECK (array_length(tags, 1) BETWEEN 1 AND 5);

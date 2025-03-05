CREATE TABLE IF NOT EXISTS authors(
  id bigserial PRIMARY KEY,
  created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  name text,
  last_name text NOT NULL,
  version integer NOT NULL DEFAULT 1
)

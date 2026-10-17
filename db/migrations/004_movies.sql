BEGIN;
CREATE TABLE movies(
  id bigserial PRIMARY KEY,
  title text NOT NULL,
  duration_minutes int NOT NULL CHECK(duration_minutes>0)
);
COMMIT;

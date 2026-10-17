BEGIN;
CREATE TABLE showtimes(
  id bigserial PRIMARY KEY,
  movie_id bigint NOT NULL REFERENCES movies,
  auditorium_id bigint NOT NULL REFERENCES auditoriums,
  starts_at timestamptz NOT NULL,
  ends_at timestamptz NOT NULL,
  price bigint NOT NULL CHECK(price>0),
  status text NOT NULL DEFAULT 'scheduled' CHECK(status IN ('scheduled','cancelled')),
  CHECK(ends_at>starts_at),
  UNIQUE(id,auditorium_id),
  EXCLUDE USING gist(auditorium_id WITH =, tstzrange(starts_at,ends_at,'[)') WITH &&)
    WHERE (status='scheduled')
);
CREATE INDEX ON showtimes(starts_at) WHERE status='scheduled';
COMMIT;

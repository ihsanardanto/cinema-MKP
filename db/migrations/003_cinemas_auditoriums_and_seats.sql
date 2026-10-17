BEGIN;
CREATE TABLE cities(id bigserial PRIMARY KEY, name text NOT NULL UNIQUE);
CREATE TABLE cinemas(
  id bigserial PRIMARY KEY,
  city_id bigint NOT NULL REFERENCES cities,
  name text NOT NULL,
  address text NOT NULL
);
CREATE TABLE auditoriums(
  id bigserial PRIMARY KEY,
  cinema_id bigint NOT NULL REFERENCES cinemas,
  name text NOT NULL,
  UNIQUE(cinema_id,name)
);
CREATE TABLE seats(
  id bigserial PRIMARY KEY,
  auditorium_id bigint NOT NULL REFERENCES auditoriums,
  label text NOT NULL,
  UNIQUE(auditorium_id,label),
  UNIQUE(id,auditorium_id)
);
COMMIT;

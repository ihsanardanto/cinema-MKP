BEGIN;
CREATE TABLE users(
  id bigserial PRIMARY KEY,
  email text NOT NULL UNIQUE CHECK(email=lower(email)),
  password_hash text NOT NULL,
  role text NOT NULL CHECK(role IN ('admin','customer')) DEFAULT 'customer'
);
CREATE TABLE sessions(
  token_hash text PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users,
  expires_at timestamptz NOT NULL
);
COMMIT;

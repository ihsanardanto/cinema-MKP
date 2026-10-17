BEGIN;
CREATE TABLE bookings(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id bigint NOT NULL REFERENCES users,
  showtime_id bigint NOT NULL REFERENCES showtimes,
  status text NOT NULL CHECK(status IN ('held','paid','expired','cancelled','refund_pending','refunded')),
  expires_at timestamptz NOT NULL,
  amount bigint NOT NULL CHECK(amount>=0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,showtime_id)
);
CREATE INDEX ON bookings(showtime_id,status);
COMMIT;

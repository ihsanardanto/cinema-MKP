BEGIN;
CREATE TABLE payments(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  booking_id uuid NOT NULL REFERENCES bookings,
  provider_reference text NOT NULL UNIQUE,
  status text NOT NULL CHECK(status IN ('pending','succeeded','failed')),
  amount bigint NOT NULL CHECK(amount>0)
);
CREATE TABLE refunds(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_id uuid NOT NULL UNIQUE REFERENCES payments,
  status text NOT NULL CHECK(status IN ('pending','succeeded','failed')),
  amount bigint NOT NULL CHECK(amount>0),
  provider_reference text UNIQUE
);
COMMIT;

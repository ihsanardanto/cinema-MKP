BEGIN;
CREATE TABLE outbox(
  id bigserial PRIMARY KEY,
  event_type text NOT NULL,
  aggregate_id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  delivered_at timestamptz
);
CREATE INDEX ON outbox(id) WHERE delivered_at IS NULL;
COMMIT;

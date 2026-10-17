BEGIN;
CREATE TABLE showtime_seats(
  showtime_id bigint NOT NULL,
  auditorium_id bigint NOT NULL,
  seat_id bigint NOT NULL,
  state text NOT NULL DEFAULT 'available' CHECK(state IN ('available','held','sold','blocked')),
  booking_id uuid,
  hold_until timestamptz,
  PRIMARY KEY(showtime_id,seat_id),
  FOREIGN KEY(showtime_id,auditorium_id) REFERENCES showtimes(id,auditorium_id),
  FOREIGN KEY(seat_id,auditorium_id) REFERENCES seats(id,auditorium_id),
  FOREIGN KEY(booking_id,showtime_id) REFERENCES bookings(id,showtime_id),
  CHECK((state='available' AND booking_id IS NULL AND hold_until IS NULL) OR
        (state='held' AND booking_id IS NOT NULL AND hold_until IS NOT NULL) OR
        (state='sold' AND booking_id IS NOT NULL AND hold_until IS NULL) OR state='blocked')
);
CREATE TABLE booking_items(
  booking_id uuid NOT NULL,
  showtime_id bigint NOT NULL,
  seat_id bigint NOT NULL,
  unit_price bigint NOT NULL CHECK(unit_price>0),
  PRIMARY KEY(booking_id,seat_id),
  FOREIGN KEY(booking_id,showtime_id) REFERENCES bookings(id,showtime_id),
  FOREIGN KEY(showtime_id,seat_id) REFERENCES showtime_seats(showtime_id,seat_id)
);
CREATE INDEX ON showtime_seats(hold_until) WHERE state='held';
COMMIT;

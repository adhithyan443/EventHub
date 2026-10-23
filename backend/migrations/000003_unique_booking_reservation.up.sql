
-- A reservation can be associated with at most one booking.
-- PostgreSQL permits multiple NULL values in a unique index.
CREATE UNIQUE INDEX uq_bookings_reservation_id
    ON bookings (reservation_id)
    WHERE reservation_id IS NOT NULL;

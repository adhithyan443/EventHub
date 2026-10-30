-- Migration: 000012_add_cancellation_snapshot_to_bookings.up.sql
-- Description: Add cancellation policy snapshot columns to bookings table with safe defaults and historical backfill.

ALTER TABLE bookings
ADD COLUMN cancellation_allowed BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN cancellation_deadline_hours INTEGER NOT NULL DEFAULT 0,
ADD COLUMN cancellation_refund_policy VARCHAR(50) NOT NULL DEFAULT 'NO_REFUND',
ADD COLUMN cancellation_refund_percentage INTEGER NOT NULL DEFAULT 0;

-- Backfill existing bookings from event_cancellations where the policy is known.
-- Records without an associated event cancellation policy safely retain the NO_REFUND defaults.
UPDATE bookings AS b
SET 
    cancellation_allowed = ec.cancellation_allowed,
    cancellation_deadline_hours = ec.cancellation_deadline_hours,
    cancellation_refund_policy = ec.refund_policy,
    cancellation_refund_percentage = ec.refund_percentage
FROM event_cancellations AS ec
WHERE b.event_id = ec.event_id;

-- Integrity constraints matching cancellation policy validation rules
ALTER TABLE bookings
ADD CONSTRAINT chk_bookings_cancellation_percentage
CHECK (cancellation_refund_percentage >= 0 AND cancellation_refund_percentage <= 100);

ALTER TABLE bookings
ADD CONSTRAINT chk_bookings_cancellation_deadline
CHECK (cancellation_deadline_hours >= 0);

-- Migration: 000012_add_cancellation_snapshot_to_bookings.down.sql
-- Description: Remove cancellation policy snapshot columns and constraints from bookings table.

ALTER TABLE bookings
DROP CONSTRAINT IF EXISTS chk_bookings_cancellation_percentage;

ALTER TABLE bookings
DROP CONSTRAINT IF EXISTS chk_bookings_cancellation_deadline;

ALTER TABLE bookings
DROP COLUMN IF EXISTS cancellation_refund_percentage,
DROP COLUMN IF EXISTS cancellation_refund_policy,
DROP COLUMN IF EXISTS cancellation_deadline_hours,
DROP COLUMN IF EXISTS cancellation_allowed;

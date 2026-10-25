-- Migration: Add partial unique index for concurrency safety on pending payments.
-- 
-- IMPORTANT: This migration contains NO automatic status mutations.
-- Before applying this migration in environments with existing data,
-- run the pre-migration audit query to ensure no duplicate PENDING records exist:
--
--   SELECT booking_id, COUNT(*) AS pending_count
--   FROM payments
--   WHERE status = 'PENDING'
--   GROUP BY booking_id
--   HAVING COUNT(*) > 1;
--
-- If any duplicates are returned, inspect them manually and resolve appropriately
-- before running this migration. If duplicate PENDING records exist, this migration
-- will safely fail and rollback without altering any historical payment records.

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_booking_pending
ON payments (booking_id)
WHERE status = 'PENDING';

ALTER TABLE tickets
DROP CONSTRAINT IF EXISTS uq_ticket_booking_item_admission;

ALTER TABLE tickets
DROP CONSTRAINT IF EXISTS chk_ticket_admission_number;

ALTER TABLE tickets
DROP COLUMN IF EXISTS admission_number;
ALTER TABLE tickets
ADD COLUMN admission_number INT NOT NULL DEFAULT 1;

ALTER TABLE tickets
ADD CONSTRAINT uq_ticket_booking_item_admission
UNIQUE (booking_item_id, admission_number);
ALTER TABLE tickets
ADD COLUMN admission_number INT;

WITH numbered_tickets AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY booking_item_id
            ORDER BY issued_at, id
        ) AS admission_number
    FROM tickets
)
UPDATE tickets AS t
SET admission_number = nt.admission_number
FROM numbered_tickets AS nt
WHERE t.id = nt.id;

ALTER TABLE tickets
ALTER COLUMN admission_number SET NOT NULL;

ALTER TABLE tickets
ADD CONSTRAINT chk_ticket_admission_number
CHECK (admission_number > 0);

ALTER TABLE tickets
ADD CONSTRAINT uq_ticket_booking_item_admission
UNIQUE (booking_item_id, admission_number);
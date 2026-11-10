-- Migration: 000015_create_invoices.up.sql
-- Description: Create invoices and invoice_items tables with sequence, constraints, and idempotent historical backfill.

-- 1. Sequence for Concurrency-Safe Receipt Numbering (Monotonic sequence; rolled-back transactions may leave gaps)
CREATE SEQUENCE IF NOT EXISTS booking_receipt_number_seq START WITH 100001 INCREMENT BY 1;

-- 2. Invoices (Booking Receipts) Table
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50) NOT NULL UNIQUE,

    booking_id UUID NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE RESTRICT,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,

    status VARCHAR(30) NOT NULL DEFAULT 'ISSUED',
    currency VARCHAR(10) NOT NULL DEFAULT 'INR',

    -- Immutable Financial Snapshot
    subtotal NUMERIC(10, 2) NOT NULL,
    platform_fee NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    tax_amount NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    discount_amount NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    total_amount NUMERIC(10, 2) NOT NULL,
    paid_amount NUMERIC(10, 2) NOT NULL,

    -- Immutable Customer Snapshot
    customer_name VARCHAR(100) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),

    -- Immutable Event & Venue Snapshot (Preserves Wall-Clock Time without Timezone Distortion)
    event_title VARCHAR(255) NOT NULL,
    event_date DATE NOT NULL,
    event_time_str VARCHAR(50),
    venue_name VARCHAR(255),
    venue_address TEXT,
    is_online BOOLEAN NOT NULL DEFAULT FALSE,
    online_details TEXT,

    -- Immutable Issuer / Organizer Snapshot
    issuer_name VARCHAR(255) NOT NULL,
    organizer_name VARCHAR(255),

    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_invoices_status CHECK (status IN ('ISSUED', 'VOID')),
    CONSTRAINT chk_invoices_subtotal CHECK (subtotal >= 0),
    CONSTRAINT chk_invoices_platform_fee CHECK (platform_fee >= 0),
    CONSTRAINT chk_invoices_tax CHECK (tax_amount >= 0),
    CONSTRAINT chk_invoices_discount CHECK (discount_amount >= 0),
    CONSTRAINT chk_invoices_total CHECK (total_amount >= 0),
    CONSTRAINT chk_invoices_paid CHECK (paid_amount >= 0)
);

CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_booking_id ON invoices(booking_id);
CREATE INDEX idx_invoices_payment_id ON invoices(payment_id);
CREATE INDEX idx_invoices_issued_at ON invoices(issued_at);

-- 3. Invoice Items Table (Immutable Ticket Line Items)
-- ON DELETE RESTRICT ensures financial line items are preserved and never silently cascade deleted.
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    booking_item_id UUID REFERENCES booking_items(id) ON DELETE SET NULL,

    category_name VARCHAR(100) NOT NULL,
    seat_label VARCHAR(100),
    quantity INTEGER NOT NULL,
    unit_price NUMERIC(10, 2) NOT NULL,
    total_price NUMERIC(10, 2) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_invoice_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_invoice_items_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_invoice_items_total_price CHECK (total_price >= 0)
);

CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);

-- Partial Unique Index for Item-Level Backfill Idempotency
CREATE UNIQUE INDEX uq_invoice_items_booking_item_id
ON invoice_items (booking_item_id)
WHERE booking_item_id IS NOT NULL;

-- =========================================================================
-- 4. IDEMPOTENT, DEFENSIVE HISTORICAL BACKFILL
-- =========================================================================

-- Step 4A: Backfill Invoices ONLY for valid paid bookings with items.
-- Excludes PENDING bookings, unpaid cancellations, and 0-item test records.
-- Preserves local wall-clock event date & time strings without UTC distortion.
INSERT INTO invoices (
    id, invoice_number, booking_id, payment_id, user_id, event_id, status, currency,
    subtotal, platform_fee, tax_amount, discount_amount, total_amount, paid_amount,
    customer_name, customer_email, customer_phone, event_title, event_date, event_time_str,
    venue_name, venue_address, is_online, online_details, issuer_name, organizer_name,
    issued_at, created_at, updated_at
)
SELECT 
    gen_random_uuid(),
    'EVH-REC-' || TO_CHAR(COALESCE(p.paid_at, b.created_at), 'YYYYMMDD') || '-' || LPAD(nextval('booking_receipt_number_seq')::text, 6, '0'),
    b.id,
    p.id,
    b.user_id,
    b.event_id,
    'ISSUED',
    'INR',
    b.subtotal,
    b.platform_fee,
    b.tax,
    b.discount,
    b.total,
    p.amount,
    COALESCE(NULLIF(TRIM(u.full_name), ''), 'Valued Customer'),
    COALESCE(u.email, ''),
    u.phone,
    e.title,
    COALESCE(sched.event_date, b.created_at::date),
    sched.time_display,
    v.name,
    v.address,
    (e.event_type = 'ONLINE'),
    e.online_url,
    'EventHub Bookings',
    COALESCE(NULLIF(TRIM(op.business_name), ''), 'Event Organizer'),
    COALESCE(p.paid_at, b.created_at),
    b.created_at,
    NOW()
FROM bookings b
JOIN (
    -- Exactly one successful payment per booking
    SELECT DISTINCT ON (booking_id) id, booking_id, amount, paid_at
    FROM payments
    WHERE status = 'SUCCESS'
    ORDER BY booking_id, paid_at DESC
) p ON p.booking_id = b.id
JOIN users u ON u.id = b.user_id
JOIN events e ON e.id = b.event_id
LEFT JOIN organizer_profiles op ON op.organizer_id = e.organizer_id
LEFT JOIN venues v ON v.id = e.venue_id
LEFT JOIN LATERAL (
    SELECT 
        es.event_date,
        CASE 
            WHEN es.is_all_day THEN 'All Day'
            WHEN es.start_time IS NOT NULL AND es.end_time IS NOT NULL THEN 
                TO_CHAR(es.start_time, 'HH12:MI AM') || ' – ' || TO_CHAR(es.end_time, 'HH12:MI AM')
            WHEN es.start_time IS NOT NULL THEN 
                TO_CHAR(es.start_time, 'HH12:MI AM')
            ELSE 'Time TBA'
        END AS time_display
    FROM event_schedules es
    WHERE es.event_id = e.id
    ORDER BY es.event_date ASC, es.start_time ASC
    LIMIT 1
) sched ON TRUE
WHERE b.status IN ('CONFIRMED', 'CANCELLED')
  AND EXISTS (SELECT 1 FROM booking_items bi WHERE bi.booking_id = b.id)
  AND NOT EXISTS (SELECT 1 FROM invoices inv WHERE inv.booking_id = b.id);

-- Step 4B: Backfill Invoice Items with Item-Level Idempotency.
-- Operates per booking_item_id; skips already-backfilled lines on rerun.
INSERT INTO invoice_items (
    id, invoice_id, booking_item_id, category_name, seat_label, quantity, unit_price, total_price, created_at
)
SELECT 
    gen_random_uuid(),
    inv.id,
    bi.id,
    COALESCE(tt.name, 'General Admission'),
    CASE WHEN s.id IS NOT NULL THEN 'Row ' || sr.row_name || ', Seat ' || s.seat_number ELSE NULL END,
    bi.quantity,
    bi.unit_price,
    bi.total_price,
    NOW()
FROM invoices inv
JOIN booking_items bi ON bi.booking_id = inv.booking_id
LEFT JOIN ticket_types tt ON tt.id = bi.ticket_type_id
LEFT JOIN seats s ON s.id = bi.seat_id
LEFT JOIN seat_rows sr ON sr.id = s.row_id
ON CONFLICT (booking_item_id) WHERE booking_item_id IS NOT NULL DO NOTHING;

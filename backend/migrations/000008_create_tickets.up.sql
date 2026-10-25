-- Create individual tickets issued for confirmed bookings.
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    ticket_code VARCHAR(100) NOT NULL UNIQUE,

    booking_id UUID NOT NULL
        REFERENCES bookings(id) ON DELETE RESTRICT,

    booking_item_id UUID NOT NULL
        REFERENCES booking_items(id) ON DELETE RESTRICT,

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    event_id UUID NOT NULL
        REFERENCES events(id) ON DELETE RESTRICT,

    ticket_type_id UUID
        REFERENCES ticket_types(id) ON DELETE RESTRICT,

    seat_id UUID
        REFERENCES seats(id) ON DELETE RESTRICT,

    price NUMERIC(10, 2) NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'VALID',

    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tickets_price CHECK (price >= 0),

    CONSTRAINT chk_tickets_status CHECK (
        status IN ('VALID', 'USED', 'CANCELLED', 'REFUNDED')
    )
);

CREATE INDEX idx_tickets_booking_id ON tickets(booking_id);
CREATE INDEX idx_tickets_booking_item_id ON tickets(booking_item_id);
CREATE INDEX idx_tickets_user_id ON tickets(user_id);
CREATE INDEX idx_tickets_event_id ON tickets(event_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_seat_id ON tickets(seat_id);
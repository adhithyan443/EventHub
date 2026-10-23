-- Create bookings table.
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    booking_number VARCHAR(50) NOT NULL UNIQUE,

    user_id UUID NOT NULL REFERENCES users(id),
    event_id UUID NOT NULL REFERENCES events(id),
    reservation_id UUID REFERENCES seat_reservations(id),

    subtotal NUMERIC(10, 2) NOT NULL DEFAULT 0,
    platform_fee NUMERIC(10, 2) NOT NULL DEFAULT 0,
    tax NUMERIC(10, 2) NOT NULL DEFAULT 0,
    discount NUMERIC(10, 2) NOT NULL DEFAULT 0,
    total NUMERIC(10, 2) NOT NULL DEFAULT 0,

    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_bookings_subtotal CHECK (subtotal >= 0),
    CONSTRAINT chk_bookings_platform_fee CHECK (platform_fee >= 0),
    CONSTRAINT chk_bookings_tax CHECK (tax >= 0),
    CONSTRAINT chk_bookings_discount CHECK (discount >= 0),
    CONSTRAINT chk_bookings_total CHECK (total >= 0)
);

CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_event_id ON bookings(event_id);
CREATE INDEX idx_bookings_reservation_id ON bookings(reservation_id);
CREATE INDEX idx_bookings_deleted_at ON bookings(deleted_at);


-- Create booking items table.
CREATE TABLE booking_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    booking_id UUID NOT NULL
        REFERENCES bookings(id) ON DELETE CASCADE,

    seat_id UUID,
    ticket_type_id UUID REFERENCES ticket_types(id),

    quantity INTEGER NOT NULL,
    unit_price NUMERIC(10, 2) NOT NULL,
    total_price NUMERIC(10, 2) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_booking_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_booking_items_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_booking_items_total_price CHECK (total_price >= 0),

    CONSTRAINT chk_booking_items_inventory CHECK (
        seat_id IS NOT NULL OR ticket_type_id IS NOT NULL
    )
);

CREATE INDEX idx_booking_items_booking_id
    ON booking_items(booking_id);

CREATE INDEX idx_booking_items_ticket_type_id
    ON booking_items(ticket_type_id);

CREATE INDEX idx_booking_items_seat_id
    ON booking_items(seat_id);
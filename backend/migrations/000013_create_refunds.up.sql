-- Migration: 000013_create_refunds.up.sql
-- Description: Create refunds table to record and track payment refunds with gateway reconciliation.

CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id),
    payment_id UUID NOT NULL REFERENCES payments(id),
    gateway_refund_id VARCHAR(100),
    amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    reason VARCHAR(255) NOT NULL DEFAULT 'CUSTOMER_CANCELLATION',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT chk_refunds_amount CHECK (amount >= 0),
    CONSTRAINT chk_refunds_status CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'))
);

CREATE INDEX idx_refunds_booking_id ON refunds(booking_id);
CREATE INDEX idx_refunds_payment_id ON refunds(payment_id);
CREATE UNIQUE INDEX uq_refunds_gateway_refund_id ON refunds(gateway_refund_id) WHERE gateway_refund_id IS NOT NULL;

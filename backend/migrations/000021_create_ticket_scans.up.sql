-- Migration: 000021_create_ticket_scans.up.sql
-- Description: Create ticket_scans table according to Section 9.2 for organizer ticket check-in logs.

CREATE TABLE IF NOT EXISTS ticket_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id),
    scanned_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL,
    scanned_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_scans_ticket_id ON ticket_scans(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_scans_scanned_by ON ticket_scans(scanned_by);
CREATE INDEX IF NOT EXISTS idx_ticket_scans_scanned_at ON ticket_scans(scanned_at DESC);

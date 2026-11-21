-- Migration: 000019_create_settlements.up.sql
-- Description: Create settlements table for organizer settlement requests and payout tracking.

CREATE TABLE IF NOT EXISTS settlements (
    id UUID PRIMARY KEY,
    organizer_id UUID NOT NULL,
    event_id UUID NULL,
    amount NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    payout_reference VARCHAR(255) NULL,
    failure_reason TEXT NULL,
    rejection_reason TEXT NULL,
    approved_by UUID NULL,
    approved_at TIMESTAMPTZ NULL,
    rejected_at TIMESTAMPTZ NULL,
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_settlements_amount CHECK (amount > 0),
    CONSTRAINT chk_settlements_currency CHECK (currency = 'INR'),
    CONSTRAINT chk_settlements_status CHECK (status IN ('PENDING', 'APPROVED', 'PROCESSING', 'COMPLETED', 'REJECTED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_settlements_organizer_id ON settlements(organizer_id);
CREATE INDEX IF NOT EXISTS idx_settlements_event_id ON settlements(event_id);
CREATE INDEX IF NOT EXISTS idx_settlements_status ON settlements(status);
CREATE INDEX IF NOT EXISTS idx_settlements_created_at ON settlements(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_settlements_payout_reference ON settlements(payout_reference) WHERE payout_reference IS NOT NULL;

-- Seed baseline chart of accounts if not present
INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
VALUES 
    ('a0000000-0000-0000-0000-000000001010', '1010', 'Razorpay Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
    ('a0000000-0000-0000-0000-000000001020', '1020', 'Operating Bank Account', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
    ('a0000000-0000-0000-0000-000000002010', '2010', 'Organizer Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
    ('a0000000-0000-0000-0000-000000002020', '2020', 'Customer Refund Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
    ('a0000000-0000-0000-0000-000000004010', '4010', 'Platform Fee Revenue', 'REVENUE', 'CREDIT', 'INR', TRUE, NOW()),
    ('a0000000-0000-0000-0000-000000005010', '5010', 'Payment Gateway Expense', 'EXPENSE', 'DEBIT', 'INR', TRUE, NOW())
ON CONFLICT (code) DO NOTHING;

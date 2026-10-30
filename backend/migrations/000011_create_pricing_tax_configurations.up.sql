-- Migration: 000011_create_pricing_tax_configurations.up.sql
-- Description: Create pricing_tax_configurations table with temporal constraints, contiguity triggers, and initial default configuration.

CREATE TABLE public.pricing_tax_configurations (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    platform_fee_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    platform_fee_percent NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    tax_percent NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_basis VARCHAR(32) NOT NULL DEFAULT 'UNVALIDATED',
    tax_rules_validated BOOLEAN NOT NULL DEFAULT FALSE,
    tax_legal_signoff_reference VARCHAR(128) NULL,
    free_ticket_exempt BOOLEAN NOT NULL DEFAULT TRUE,
    effective_from TIMESTAMP WITH TIME ZONE NOT NULL,
    effective_to TIMESTAMP WITH TIME ZONE NULL,
    cancelled_at TIMESTAMP WITH TIME ZONE NULL,
    cancelled_by UUID NULL REFERENCES public.users(id) ON DELETE SET NULL,
    created_by UUID NULL REFERENCES public.users(id) ON DELETE SET NULL,
    change_reason VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,

    CONSTRAINT chk_effective_bounds 
        CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT chk_platform_fee_percent 
        CHECK (platform_fee_percent >= 0.00 AND platform_fee_percent <= 100.00),
    CONSTRAINT chk_tax_percent 
        CHECK (tax_percent >= 0.00 AND tax_percent <= 100.00),
    CONSTRAINT chk_tax_basis_values 
        CHECK (tax_basis IN ('UNVALIDATED', 'PLATFORM_FEE_ONLY', 'SUBTOTAL_ONLY', 'TOTAL_BEFORE_TAX')),
    CONSTRAINT chk_status_values 
        CHECK (status IN ('ACTIVE', 'SCHEDULED', 'SUPERSEDED', 'CANCELLED'))
);

-- Partial deferrable exclusion constraint to prevent overlapping active or scheduled configurations
ALTER TABLE public.pricing_tax_configurations
ADD CONSTRAINT uq_no_overlapping_pricing_tax_intervals
EXCLUDE USING gist (
    tstzrange(effective_from, COALESCE(effective_to, 'infinity'::timestamptz), '[)') WITH &&
)
WHERE (status != 'CANCELLED')
DEFERRABLE INITIALLY DEFERRED;

-- Contiguity verification trigger function
CREATE OR REPLACE FUNCTION fn_verify_pricing_tax_contiguity()
RETURNS trigger AS $$
DECLARE
    gap_count INTEGER;
    premature_tail_count INTEGER;
BEGIN
    WITH ordered AS (
        SELECT effective_from, effective_to,
               LEAD(effective_from) OVER (ORDER BY effective_from) AS next_from
        FROM public.pricing_tax_configurations
        WHERE status != 'CANCELLED'
    )
    SELECT COUNT(*) INTO gap_count
    FROM ordered
    WHERE next_from IS NOT NULL AND (effective_to IS NULL OR effective_to <> next_from);

    IF gap_count > 0 THEN
        RAISE EXCEPTION 'Pricing timeline contiguity violation: gap or unclosed intermediate interval detected'
            USING ERRCODE = '23514';
    END IF;

    SELECT COUNT(*) INTO premature_tail_count
    FROM (
        SELECT effective_to
        FROM public.pricing_tax_configurations
        WHERE status != 'CANCELLED'
        ORDER BY effective_from DESC
        LIMIT 1
    ) tail
    WHERE tail.effective_to IS NOT NULL;

    IF premature_tail_count > 0 THEN
        RAISE EXCEPTION 'Pricing timeline must terminate in an open-ended interval (effective_to IS NULL)'
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_verify_pricing_tax_contiguity
AFTER INSERT OR UPDATE OR DELETE ON public.pricing_tax_configurations
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_verify_pricing_tax_contiguity();

-- Fast lookup index for active configuration by timestamp
CREATE INDEX idx_pricing_tax_configs_effective_range 
ON public.pricing_tax_configurations (effective_from, effective_to)
WHERE (status != 'CANCELLED');

-- Seed Version 1: Initial default configuration (disabled, 0.00%, open-ended from epoch)
INSERT INTO public.pricing_tax_configurations (
    id, version, status,
    platform_fee_enabled, platform_fee_percent,
    tax_enabled, tax_percent, tax_basis, tax_rules_validated, tax_legal_signoff_reference,
    free_ticket_exempt,
    effective_from, effective_to,
    change_reason, created_at, updated_at
) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    1, 'ACTIVE',
    FALSE, 0.00,
    FALSE, 0.00, 'UNVALIDATED', FALSE, NULL,
    TRUE,
    '1970-01-01 00:00:00+00', NULL,
    'Initial default configuration (disabled, 0.00%)',
    NOW(), NOW()
);

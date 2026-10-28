-- Migration: 000011_create_pricing_tax_configurations.down.sql
-- Description: Drop pricing_tax_configurations table, trigger, and functions.
-- Note: Do NOT drop shared extensions.

DROP TABLE IF EXISTS public.pricing_tax_configurations CASCADE;
DROP FUNCTION IF EXISTS public.fn_verify_pricing_tax_contiguity();

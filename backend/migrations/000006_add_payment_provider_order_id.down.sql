DROP INDEX IF EXISTS idx_payments_provider_order_id;

ALTER TABLE payments
DROP COLUMN IF EXISTS provider_order_id;
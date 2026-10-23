ALTER TABLE payments
ADD COLUMN provider_order_id VARCHAR(100);

CREATE UNIQUE INDEX idx_payments_provider_order_id
ON payments (provider_order_id)
WHERE provider_order_id IS NOT NULL;
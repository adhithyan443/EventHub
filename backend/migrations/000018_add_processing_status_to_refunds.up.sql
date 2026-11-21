-- Migration: 000018_add_processing_status_to_refunds.up.sql
-- Description: Update refunds status check constraint to support PROCESSING state.

ALTER TABLE refunds DROP CONSTRAINT IF EXISTS chk_refunds_status;
ALTER TABLE refunds ADD CONSTRAINT chk_refunds_status CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCESS', 'FAILED'));

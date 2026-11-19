-- Migration: 000018_add_processing_status_to_refunds.down.sql
-- Description: Revert refunds status check constraint back to PENDING, SUCCESS, FAILED.

UPDATE refunds SET status = 'PENDING' WHERE status = 'PROCESSING';
ALTER TABLE refunds DROP CONSTRAINT IF EXISTS chk_refunds_status;
ALTER TABLE refunds ADD CONSTRAINT chk_refunds_status CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'));

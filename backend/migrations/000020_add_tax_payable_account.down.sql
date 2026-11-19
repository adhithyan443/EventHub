-- Migration: 000020_add_tax_payable_account.down.sql
-- Description: Rollback Tax Payable (2030) account.

DELETE FROM finance_accounts WHERE code = '2030';

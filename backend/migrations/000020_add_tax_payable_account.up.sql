-- Migration: 000020_add_tax_payable_account.up.sql
-- Description: Seed Tax Payable (2030) account in finance_accounts chart of accounts.

INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
VALUES ('a0000000-0000-0000-0000-000000002030', '2030', 'Tax Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW())
ON CONFLICT (code) DO NOTHING;

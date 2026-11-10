-- Migration: 000015_create_invoices.down.sql
-- Description: Drop invoice_items and invoices tables and booking_receipt_number_seq.

DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP SEQUENCE IF EXISTS booking_receipt_number_seq;

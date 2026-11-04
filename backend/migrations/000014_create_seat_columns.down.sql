-- Migration: 000014_create_seat_columns.down.sql
-- Description: Drop seat_columns table and remove column fields from seats table.

DROP INDEX IF EXISTS idx_seats_column_id;
ALTER TABLE seats DROP COLUMN IF EXISTS column_id;
ALTER TABLE seats DROP COLUMN IF EXISTS grid_col;
DROP TABLE IF EXISTS seat_columns;

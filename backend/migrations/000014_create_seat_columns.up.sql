-- Migration: 000014_create_seat_columns.up.sql
-- Description: Create seat_columns table and add column positioning to seats table.

CREATE TABLE seat_columns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seat_layout_id UUID NOT NULL REFERENCES seat_layouts(id) ON DELETE CASCADE,
    label VARCHAR(50) NOT NULL,
    order_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_seat_columns_layout_order UNIQUE (seat_layout_id, order_index),
    CONSTRAINT uq_seat_columns_layout_label UNIQUE (seat_layout_id, label)
);

CREATE INDEX idx_seat_columns_seat_layout_id ON seat_columns(seat_layout_id);

ALTER TABLE seats
    ADD COLUMN column_id UUID NULL REFERENCES seat_columns(id) ON DELETE SET NULL,
    ADD COLUMN grid_col INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_seats_column_id ON seats(column_id);

-- Backfill existing seat layouts with default alphabetic columns (A, B, C...)
DO $$
DECLARE
    layout_rec RECORD;
    max_seats INTEGER;
    col_idx INTEGER;
    col_label VARCHAR(50);
    new_col_id UUID;
BEGIN
    FOR layout_rec IN SELECT id FROM seat_layouts LOOP
        SELECT COALESCE(MAX(s.seat_number), 0) INTO max_seats
        FROM seats s
        JOIN seat_rows sr ON sr.id = s.row_id
        JOIN seat_sections ss ON ss.id = sr.section_id
        WHERE ss.seat_layout_id = layout_rec.id;

        IF max_seats > 0 THEN
            FOR col_idx IN 0..(max_seats - 1) LOOP
                new_col_id := gen_random_uuid();
                IF col_idx < 26 THEN
                    col_label := CHR(65 + col_idx);
                ELSE
                    col_label := 'C' || (col_idx + 1);
                END IF;

                INSERT INTO seat_columns (id, seat_layout_id, label, order_index)
                VALUES (new_col_id, layout_rec.id, col_label, col_idx);

                UPDATE seats s
                SET column_id = new_col_id, grid_col = col_idx
                FROM seat_rows sr
                JOIN seat_sections ss ON ss.id = sr.section_id
                WHERE s.row_id = sr.id
                  AND ss.seat_layout_id = layout_rec.id
                  AND s.seat_number = (col_idx + 1);
            END LOOP;
        END IF;
    END LOOP;
END $$;

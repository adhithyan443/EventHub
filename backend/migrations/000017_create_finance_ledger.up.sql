CREATE TABLE finance_accounts (
    id UUID PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    normal_balance VARCHAR(10) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_finance_accounts_type
        CHECK (type IN ('ASSET', 'LIABILITY', 'REVENUE', 'EXPENSE', 'EQUITY')),

    CONSTRAINT chk_finance_accounts_normal_balance
        CHECK (normal_balance IN ('DEBIT', 'CREDIT')),

    CONSTRAINT chk_finance_accounts_normal_balance_by_type
        CHECK (
            (type IN ('ASSET', 'EXPENSE') AND normal_balance = 'DEBIT')
            OR
            (type IN ('LIABILITY', 'REVENUE', 'EQUITY') AND normal_balance = 'CREDIT')
        ),

    CONSTRAINT chk_finance_accounts_currency
        CHECK (currency = 'INR')
);

-- Journal headers
CREATE TABLE finance_journals (
    id UUID PRIMARY KEY,
    transaction_type VARCHAR(50) NOT NULL,
    financial_event_type VARCHAR(50) NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(255) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    idempotency_fingerprint CHAR(64) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_at TIMESTAMPTZ NULL,
    is_reversed BOOLEAN NOT NULL DEFAULT FALSE,
    reversal_reference_id UUID NULL,
    reversed_by_journal_id UUID NULL,
    reversed_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_finance_journals_reversal_ref
        FOREIGN KEY (reversal_reference_id)
        REFERENCES finance_journals(id)
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT fk_finance_journals_reversed_by
        FOREIGN KEY (reversed_by_journal_id)
        REFERENCES finance_journals(id)
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT chk_finance_journals_status
        CHECK (status IN ('DRAFT', 'POSTED')),

    CONSTRAINT chk_finance_journals_currency
        CHECK (currency = 'INR'),

    CONSTRAINT chk_finance_journals_lifecycle_timestamps
        CHECK (
            (status = 'DRAFT' AND posted_at IS NULL)
            OR
            (status = 'POSTED' AND posted_at IS NOT NULL AND posted_at >= created_at)
        ),

    CONSTRAINT chk_finance_journals_no_self_reversal
        CHECK (reversal_reference_id IS NULL OR reversal_reference_id <> id),

    CONSTRAINT chk_finance_journals_no_self_reversed_by
        CHECK (reversed_by_journal_id IS NULL OR reversed_by_journal_id <> id),

    CONSTRAINT uq_finance_journals_idempotency_key
        UNIQUE (idempotency_key),

    -- Required by the composite FK from finance_idempotency_keys.
    CONSTRAINT uq_finance_journals_composite
        UNIQUE (id, idempotency_key, idempotency_fingerprint)
);

CREATE UNIQUE INDEX idx_finance_journals_unique_reversal_ref
    ON finance_journals (reversal_reference_id)
    WHERE reversal_reference_id IS NOT NULL;

CREATE UNIQUE INDEX idx_finance_journals_unique_reversed_by
    ON finance_journals (reversed_by_journal_id)
    WHERE reversed_by_journal_id IS NOT NULL;

-- Idempotency records
CREATE TABLE finance_idempotency_keys (
    idempotency_key VARCHAR(255) NOT NULL,
    fingerprint CHAR(64) NOT NULL,
    journal_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_finance_idempotency_keys
        PRIMARY KEY (idempotency_key),

    CONSTRAINT uq_finance_idempotency_keys_journal_id
        UNIQUE (journal_id),

    CONSTRAINT fk_finance_idempotency_keys_journal_composite
        FOREIGN KEY (journal_id, idempotency_key, fingerprint)
        REFERENCES finance_journals(id, idempotency_key, idempotency_fingerprint)
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED
);

-- Double-entry ledger lines
CREATE TABLE finance_ledger_entries (
    id UUID PRIMARY KEY,
    journal_id UUID NOT NULL,
    account_id UUID NOT NULL,
    entry_type VARCHAR(10) NOT NULL,
    amount NUMERIC(20, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    organizer_id UUID NULL,
    event_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_finance_ledger_entries_journal
        FOREIGN KEY (journal_id)
        REFERENCES finance_journals(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_finance_ledger_entries_account
        FOREIGN KEY (account_id)
        REFERENCES finance_accounts(id)
        ON DELETE RESTRICT,

    CONSTRAINT chk_finance_ledger_entries_entry_type
        CHECK (entry_type IN ('DEBIT', 'CREDIT')),

    CONSTRAINT chk_finance_ledger_entries_positive_amount
        CHECK (amount > 0),

    CONSTRAINT chk_finance_ledger_entries_currency
        CHECK (currency = 'INR')
);

CREATE INDEX idx_finance_ledger_entries_journal_id
    ON finance_ledger_entries (journal_id);

CREATE INDEX idx_finance_ledger_entries_account_id
    ON finance_ledger_entries (account_id);

CREATE INDEX idx_finance_ledger_entries_organizer_id
    ON finance_ledger_entries (organizer_id);

CREATE INDEX idx_finance_ledger_entries_event_id
    ON finance_ledger_entries (event_id);


-- 1. Prevent mutations to ledger entries belonging to posted journals.
CREATE OR REPLACE FUNCTION fn_prevent_posted_entry_mutation()
RETURNS TRIGGER AS $$
DECLARE
    v_journal_status VARCHAR(20);
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.journal_id <> NEW.journal_id THEN
            RAISE EXCEPTION 'journal_id on ledger entries is strictly immutable'
                USING ERRCODE = 'P0001';
        END IF;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        SELECT status INTO v_journal_status
        FROM finance_journals
        WHERE id = NEW.journal_id
        FOR SHARE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'parent journal transaction % does not exist', NEW.journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_journal_status = 'POSTED' THEN
            RAISE EXCEPTION 'cannot insert or update ledger entry for already POSTED journal %', NEW.journal_id
                USING ERRCODE = 'P0001';
        END IF;

        RETURN NEW;
    END IF;

    IF TG_OP = 'DELETE' THEN
        SELECT status INTO v_journal_status
        FROM finance_journals
        WHERE id = OLD.journal_id
        FOR SHARE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'parent journal transaction % does not exist', OLD.journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_journal_status = 'POSTED' THEN
            RAISE EXCEPTION 'cannot delete ledger entry from already POSTED journal %', OLD.journal_id
                USING ERRCODE = 'P0001';
        END IF;

        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_posted_entry_mutation
BEFORE INSERT OR UPDATE OR DELETE ON finance_ledger_entries
FOR EACH ROW
EXECUTE FUNCTION fn_prevent_posted_entry_mutation();


-- 2. Prevent mutation or deletion of posted journals.
CREATE OR REPLACE FUNCTION fn_prevent_posted_journal_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'POSTED' THEN
            RAISE EXCEPTION 'cannot delete POSTED journal transaction %', OLD.id
                USING ERRCODE = 'P0001';
        END IF;

        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'POSTED' THEN
            IF NEW.status IS DISTINCT FROM 'POSTED' THEN
                RAISE EXCEPTION 'cannot change status of POSTED journal transaction %', OLD.id
                    USING ERRCODE = 'P0001';
            END IF;

            -- Immutable financial and identity fields.
            IF NEW.id IS DISTINCT FROM OLD.id OR
               NEW.transaction_type IS DISTINCT FROM OLD.transaction_type OR
               NEW.financial_event_type IS DISTINCT FROM OLD.financial_event_type OR
               NEW.source_type IS DISTINCT FROM OLD.source_type OR
               NEW.source_id IS DISTINCT FROM OLD.source_id OR
               NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key OR
               NEW.idempotency_fingerprint IS DISTINCT FROM OLD.idempotency_fingerprint OR
               NEW.currency IS DISTINCT FROM OLD.currency OR
               NEW.description IS DISTINCT FROM OLD.description OR
               NEW.reversal_reference_id IS DISTINCT FROM OLD.reversal_reference_id OR
               NEW.created_at IS DISTINCT FROM OLD.created_at OR
               NEW.posted_at IS DISTINCT FROM OLD.posted_at THEN
                RAISE EXCEPTION 'cannot update immutable fields of POSTED journal transaction %', OLD.id
                    USING ERRCODE = 'P0001';
            END IF;

            -- Once reversed, reversal metadata is immutable.
            IF OLD.is_reversed = TRUE THEN
                IF NEW.is_reversed IS DISTINCT FROM OLD.is_reversed OR
                   NEW.reversed_by_journal_id IS DISTINCT FROM OLD.reversed_by_journal_id OR
                   NEW.reversed_at IS DISTINCT FROM OLD.reversed_at THEN
                    RAISE EXCEPTION 'cannot modify reversal metadata on already reversed journal %', OLD.id
                        USING ERRCODE = 'P0001';
                END IF;
            END IF;

            -- Reversal metadata must transition atomically and only once.
            IF OLD.is_reversed = FALSE THEN
                IF NEW.is_reversed IS TRUE OR
                   NEW.reversed_by_journal_id IS NOT NULL OR
                   NEW.reversed_at IS NOT NULL THEN

                    IF NEW.is_reversed IS NOT TRUE OR
                       NEW.reversed_by_journal_id IS NULL OR
                       NEW.reversed_at IS NULL THEN
                        RAISE EXCEPTION
                            'reversal metadata transition on journal % must set is_reversed = TRUE, non-null reversed_by_journal_id, and non-null reversed_at simultaneously',
                            OLD.id
                            USING ERRCODE = 'P0001';
                    END IF;
                END IF;
            END IF;
        END IF;

        RETURN NEW;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_posted_journal_mutation
BEFORE UPDATE OR DELETE ON finance_journals
FOR EACH ROW
EXECUTE FUNCTION fn_prevent_posted_journal_mutation();

-- 3. Deferred double-entry balance and account validation.
CREATE OR REPLACE FUNCTION fn_validate_journal_double_entry()
RETURNS TRIGGER AS $$
DECLARE
    v_journal_id UUID;
    v_journal_status VARCHAR(20);
    v_journal_currency VARCHAR(3);
    v_distinct_accounts UUID[];
    v_account_record RECORD;
    v_locked_account_count INT := 0;
    v_balance NUMERIC(20, 2);
    v_entry_count INT;
    v_debit_count INT;
    v_credit_count INT;
BEGIN
    v_journal_id := NEW.id;

    -- Re-query the latest persisted journal state at commit time.
    SELECT status, currency
    INTO v_journal_status, v_journal_currency
    FROM finance_journals
    WHERE id = v_journal_id;

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- Draft journals are not subject to posting validation.
    IF v_journal_status <> 'POSTED' THEN
        RETURN NULL;
    END IF;

    SELECT ARRAY_AGG(DISTINCT account_id)
    INTO v_distinct_accounts
    FROM finance_ledger_entries
    WHERE journal_id = v_journal_id;

    IF v_distinct_accounts IS NULL
       OR ARRAY_LENGTH(v_distinct_accounts, 1) = 0 THEN
        RAISE EXCEPTION 'posted journal % has no ledger entries', v_journal_id
            USING ERRCODE = 'P0001';
    END IF;

    -- Lock referenced accounts in deterministic order.
    FOR v_account_record IN
        SELECT id, is_active, currency
        FROM finance_accounts
        WHERE id = ANY(v_distinct_accounts)
        ORDER BY id ASC
        FOR SHARE
    LOOP
        v_locked_account_count := v_locked_account_count + 1;

        IF NOT v_account_record.is_active THEN
            RAISE EXCEPTION 'account % is inactive; cannot post journal',
                v_account_record.id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_account_record.currency <> v_journal_currency THEN
            RAISE EXCEPTION
                'account % currency % does not match journal currency %',
                v_account_record.id,
                v_account_record.currency,
                v_journal_currency
                USING ERRCODE = 'P0001';
        END IF;
    END LOOP;

    IF v_locked_account_count <> ARRAY_LENGTH(v_distinct_accounts, 1) THEN
        RAISE EXCEPTION
            'one or more referenced accounts in journal % do not exist',
            v_journal_id
            USING ERRCODE = 'P0001';
    END IF;

    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE entry_type = 'DEBIT'),
        COUNT(*) FILTER (WHERE entry_type = 'CREDIT'),
        COALESCE(
            SUM(
                CASE
                    WHEN entry_type = 'DEBIT' THEN amount
                    ELSE -amount
                END
            ),
            0
        )
    INTO
        v_entry_count,
        v_debit_count,
        v_credit_count,
        v_balance
    FROM finance_ledger_entries
    WHERE journal_id = v_journal_id;

    IF v_entry_count < 2 THEN
        RAISE EXCEPTION
            'journal % must have at least 2 ledger entries, found %',
            v_journal_id,
            v_entry_count
            USING ERRCODE = 'P0001';
    END IF;

    IF v_debit_count = 0 THEN
        RAISE EXCEPTION 'journal % has no DEBIT entries', v_journal_id
            USING ERRCODE = 'P0001';
    END IF;

    IF v_credit_count = 0 THEN
        RAISE EXCEPTION 'journal % has no CREDIT entries', v_journal_id
            USING ERRCODE = 'P0001';
    END IF;

    IF v_balance <> 0 THEN
        RAISE EXCEPTION
            'unbalanced journal transaction %: balance delta is %',
            v_journal_id,
            v_balance
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_validate_journal_double_entry
AFTER INSERT OR UPDATE ON finance_journals
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_validate_journal_double_entry();

-- 4. Validate bidirectional reversal consistency at transaction commit.
CREATE OR REPLACE FUNCTION fn_validate_reversal_mutual_consistency()
RETURNS TRIGGER AS $$
DECLARE
    v_journal_id UUID;
    v_orig_journal RECORD;
    v_rev_journal RECORD;
    v_current_tx_type VARCHAR(50);
    v_current_status VARCHAR(20);
    v_current_is_reversed BOOLEAN;
    v_current_rev_ref UUID;
    v_current_reversed_by UUID;
    v_current_posted_at TIMESTAMPTZ;
    v_current_reversed_at TIMESTAMPTZ;
BEGIN
    v_journal_id := NEW.id;

    -- Re-query persisted values to validate the final state at commit.
    SELECT
        transaction_type,
        status,
        is_reversed,
        reversal_reference_id,
        reversed_by_journal_id,
        posted_at,
        reversed_at
    INTO
        v_current_tx_type,
        v_current_status,
        v_current_is_reversed,
        v_current_rev_ref,
        v_current_reversed_by,
        v_current_posted_at,
        v_current_reversed_at
    FROM finance_journals
    WHERE id = v_journal_id;

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- Case 1: Validate a reversal journal.
    IF v_current_tx_type = 'REVERSAL' THEN
        IF v_current_rev_ref IS NULL THEN
            RAISE EXCEPTION
                'reversal journal % must have a non-null reversal_reference_id',
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_current_status <> 'POSTED' THEN
            RAISE EXCEPTION
                'reversal journal % must be in POSTED status at commit, found %',
                v_journal_id,
                v_current_status
                USING ERRCODE = 'P0001';
        END IF;

        SELECT
            id,
            status,
            is_reversed,
            reversed_by_journal_id,
            posted_at,
            reversed_at
        INTO v_orig_journal
        FROM finance_journals
        WHERE id = v_current_rev_ref;

        IF NOT FOUND THEN
            RAISE EXCEPTION
                'original journal % referenced by reversal % not found',
                v_current_rev_ref,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_orig_journal.status <> 'POSTED' THEN
            RAISE EXCEPTION
                'original journal % must be POSTED to be reversed, found %',
                v_orig_journal.id,
                v_orig_journal.status
                USING ERRCODE = 'P0001';
        END IF;

        IF NOT v_orig_journal.is_reversed THEN
            RAISE EXCEPTION
                'original journal % referenced by reversal % must be marked is_reversed = TRUE',
                v_orig_journal.id,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_orig_journal.reversed_by_journal_id IS DISTINCT FROM v_journal_id THEN
            RAISE EXCEPTION
                'mutual reference mismatch: original journal % has reversed_by_journal_id %, expected %',
                v_orig_journal.id,
                v_orig_journal.reversed_by_journal_id,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_orig_journal.reversed_at IS DISTINCT FROM v_current_posted_at THEN
            RAISE EXCEPTION
                'reversal timestamp mismatch between original journal % and reversal %',
                v_orig_journal.id,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;
    END IF;

    -- Case 2: Validate an original journal marked as reversed.
    IF v_current_is_reversed = TRUE THEN
        IF v_current_status <> 'POSTED' THEN
            RAISE EXCEPTION
                'draft journal % cannot be marked reversed',
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_current_reversed_by IS NULL THEN
            RAISE EXCEPTION
                'reversed journal % must specify reversed_by_journal_id',
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        SELECT
            id,
            transaction_type,
            status,
            reversal_reference_id,
            posted_at
        INTO v_rev_journal
        FROM finance_journals
        WHERE id = v_current_reversed_by;

        IF NOT FOUND THEN
            RAISE EXCEPTION
                'reversing journal % not found for reversed journal %',
                v_current_reversed_by,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_rev_journal.transaction_type <> 'REVERSAL' THEN
            RAISE EXCEPTION
                'reversing journal % must have transaction_type REVERSAL, found %',
                v_rev_journal.id,
                v_rev_journal.transaction_type
                USING ERRCODE = 'P0001';
        END IF;

        IF v_rev_journal.status <> 'POSTED' THEN
            RAISE EXCEPTION
                'reversing journal % must be in POSTED status at commit, found %',
                v_rev_journal.id,
                v_rev_journal.status
                USING ERRCODE = 'P0001';
        END IF;

        IF v_rev_journal.reversal_reference_id IS DISTINCT FROM v_journal_id THEN
            RAISE EXCEPTION
                'mutual reference mismatch: reversing journal % has reversal_reference_id %, expected %',
                v_rev_journal.id,
                v_rev_journal.reversal_reference_id,
                v_journal_id
                USING ERRCODE = 'P0001';
        END IF;

        IF v_current_reversed_at IS DISTINCT FROM v_rev_journal.posted_at THEN
            RAISE EXCEPTION
                'reversal timestamp mismatch: original reversed_at does not match reversal posted_at'
                USING ERRCODE = 'P0001';
        END IF;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_validate_reversal_mutual_consistency
AFTER INSERT OR UPDATE ON finance_journals
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_validate_reversal_mutual_consistency();

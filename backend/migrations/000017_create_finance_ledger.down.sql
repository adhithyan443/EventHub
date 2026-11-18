-- Remove finance ledger triggers.
DROP TRIGGER IF EXISTS trg_validate_reversal_mutual_consistency
    ON finance_journals;

DROP TRIGGER IF EXISTS trg_validate_journal_double_entry
    ON finance_journals;

DROP TRIGGER IF EXISTS trg_prevent_posted_journal_mutation
    ON finance_journals;

DROP TRIGGER IF EXISTS trg_prevent_posted_entry_mutation
    ON finance_ledger_entries;


-- Remove finance ledger trigger functions.
DROP FUNCTION IF EXISTS fn_validate_reversal_mutual_consistency();

DROP FUNCTION IF EXISTS fn_validate_journal_double_entry();

DROP FUNCTION IF EXISTS fn_prevent_posted_journal_mutation();

DROP FUNCTION IF EXISTS fn_prevent_posted_entry_mutation();


-- Drop tables in dependency order.
DROP TABLE IF EXISTS finance_ledger_entries;

DROP TABLE IF EXISTS finance_idempotency_keys;

DROP TABLE IF EXISTS finance_journals;

DROP TABLE IF EXISTS finance_accounts;
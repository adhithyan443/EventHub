package repository_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminLedgerTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	ddl := `
	CREATE TABLE IF NOT EXISTS finance_accounts (
		id UUID PRIMARY KEY,
		code VARCHAR(50) NOT NULL UNIQUE,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		normal_balance VARCHAR(10) NOT NULL,
		currency VARCHAR(3) NOT NULL DEFAULT 'INR',
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS finance_journals (
		id UUID PRIMARY KEY,
		transaction_type VARCHAR(50) NOT NULL,
		financial_event_type VARCHAR(50) NOT NULL,
		source_type VARCHAR(50) NOT NULL,
		source_id VARCHAR(255) NOT NULL,
		idempotency_key VARCHAR(255) NOT NULL UNIQUE,
		idempotency_fingerprint CHAR(64) NOT NULL,
		currency VARCHAR(3) NOT NULL DEFAULT 'INR',
		description TEXT NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		posted_at TIMESTAMPTZ,
		is_reversed BOOLEAN NOT NULL DEFAULT FALSE,
		reversal_reference_id UUID,
		reversed_by_journal_id UUID,
		reversed_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS finance_ledger_entries (
		id UUID PRIMARY KEY,
		journal_id UUID NOT NULL,
		account_id UUID NOT NULL,
		entry_type VARCHAR(10) NOT NULL,
		amount NUMERIC(20,2) NOT NULL,
		currency VARCHAR(3) NOT NULL DEFAULT 'INR',
		organizer_id UUID,
		event_id UUID,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_fle_journal_id ON finance_ledger_entries(journal_id);
	`

	require.NoError(t, db.Exec(ddl).Error)

	truncateSQL := `
		TRUNCATE TABLE 
			finance_ledger_entries, 
			finance_journals, 
			finance_accounts 
		CASCADE;
	`
	require.NoError(t, db.Exec(truncateSQL).Error)
}

func TestAdminLedgerRepository_ListJournals_And_GetByID(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminLedgerTestTables(t, db)
	repo := repository.NewAdminLedgerRepository(db, slog.Default())
	ctx := context.Background()

	// 1. Empty database test
	emptyRes, err := repo.ListJournals(ctx, domain.AdminLedgerFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), emptyRes.TotalRecords)
	assert.Equal(t, 1, emptyRes.TotalPages)
	assert.Empty(t, emptyRes.Journals)

	notFoundDetail, err := repo.GetJournalByID(ctx, uuid.New())
	assert.Nil(t, notFoundDetail)
	assert.ErrorIs(t, err, domain.ErrLedgerJournalNotFound)

	// 2. Seed Accounts
	acc1010 := uuid.New()
	acc2010 := uuid.New()
	acc4010 := uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
		VALUES 
			(?, '1010', 'Razorpay Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
			(?, '2010', 'Organizer Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '4010', 'Platform Fee Revenue', 'REVENUE', 'CREDIT', 'INR', TRUE, NOW())
	`, acc1010, acc2010, acc4010).Error)

	// 3. Seed Journals:
	// Journal 1: PAYMENT - captured ticket sale (1000 INR)
	j1ID := uuid.New()
	t1 := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 'pay_12345', 'idem_pay_12345', 'fp1', 'INR', 'Ticket sale capture', 'DRAFT', ?, FALSE)
	`, j1ID, t1).Error)

	// Entries for Journal 1:
	// Debit 1010 (1000), Credit 2010 (900), Credit 4010 (100)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 1000.00, 'INR', ?),
			(?, ?, ?, 'CREDIT', 900.00, 'INR', ?),
			(?, ?, ?, 'CREDIT', 100.00, 'INR', ?)
	`, uuid.New(), j1ID, acc1010, t1,
		uuid.New(), j1ID, acc2010, t1,
		uuid.New(), j1ID, acc4010, t1).Error)

	require.NoError(t, db.Exec(`
		UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
	`, t1, j1ID).Error)

	// Journal 2: REFUND - customer cancellation (200 INR)
	j2ID := uuid.New()
	t2 := time.Date(2026, 11, 12, 14, 0, 0, 0, time.UTC)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'REFUND', 'CUSTOMER_REFUND', 'REFUND', 'rfnd_998877', 'idem_rfnd_998877', 'fp2', 'INR', 'Customer refund', 'DRAFT', ?, FALSE)
	`, j2ID, t2).Error)

	// Entries for Journal 2: Debit 2010 (200), Credit 1010 (200)
	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 200.00, 'INR', ?),
			(?, ?, ?, 'CREDIT', 200.00, 'INR', ?)
	`, uuid.New(), j2ID, acc2010, t2,
		uuid.New(), j2ID, acc1010, t2).Error)

	require.NoError(t, db.Exec(`
		UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
	`, t2, j2ID).Error)

	// 4. Test ListJournals without filters
	resAll, err := repo.ListJournals(ctx, domain.AdminLedgerFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resAll.TotalRecords)
	assert.Len(t, resAll.Journals, 2)
	// Ordered by posted_at DESC -> Journal 2 first, then Journal 1
	assert.Equal(t, j2ID, resAll.Journals[0].ID)
	assert.Equal(t, j1ID, resAll.Journals[1].ID)
	assert.True(t, resAll.Journals[0].IsBalanced)
	assert.True(t, resAll.Journals[1].IsBalanced)
	assert.True(t, resAll.Journals[1].TotalDebit.Equal(decimal.NewFromInt(1000)))
	assert.True(t, resAll.Journals[1].TotalCredit.Equal(decimal.NewFromInt(1000)))

	// 5. Test filter by transaction_type
	resPay, err := repo.ListJournals(ctx, domain.AdminLedgerFilter{Page: 1, Limit: 10, TransactionType: "PAYMENT"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resPay.TotalRecords)
	assert.Equal(t, j1ID, resPay.Journals[0].ID)

	// 6. Test filter by search (source_id)
	resSearch, err := repo.ListJournals(ctx, domain.AdminLedgerFilter{Page: 1, Limit: 10, Search: "rfnd_998877"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resSearch.TotalRecords)
	assert.Equal(t, j2ID, resSearch.Journals[0].ID)

	// 7. Test filter by date range
	fromDate := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	resDate, err := repo.ListJournals(ctx, domain.AdminLedgerFilter{Page: 1, Limit: 10, From: &fromDate, To: &toDate})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resDate.TotalRecords)
	assert.Equal(t, j2ID, resDate.Journals[0].ID)

	// 8. Test GetJournalByID
	detail1, err := repo.GetJournalByID(ctx, j1ID)
	require.NoError(t, err)
	require.NotNil(t, detail1)
	assert.Equal(t, j1ID, detail1.ID)
	assert.Equal(t, "PAYMENT", detail1.TransactionType)
	assert.Equal(t, "pay_12345", detail1.SourceID)
	assert.True(t, detail1.IsBalanced)
	assert.True(t, detail1.TotalDebit.Equal(decimal.NewFromInt(1000)))
	assert.True(t, detail1.TotalCredit.Equal(decimal.NewFromInt(1000)))
	require.Len(t, detail1.Entries, 3)

	// Verify account resolution on entries
	var found1010, found2010, found4010 bool
	for _, entry := range detail1.Entries {
		switch entry.AccountCode {
		case "1010":
			found1010 = true
			assert.Equal(t, "Razorpay Clearing", entry.AccountName)
			assert.Equal(t, "DEBIT", entry.EntryType)
			assert.True(t, entry.Amount.Equal(decimal.NewFromInt(1000)))
		case "2010":
			found2010 = true
			assert.Equal(t, "Organizer Payable", entry.AccountName)
			assert.Equal(t, "CREDIT", entry.EntryType)
			assert.True(t, entry.Amount.Equal(decimal.NewFromInt(900)))
		case "4010":
			found4010 = true
			assert.Equal(t, "Platform Fee Revenue", entry.AccountName)
			assert.Equal(t, "CREDIT", entry.EntryType)
			assert.True(t, entry.Amount.Equal(decimal.NewFromInt(100)))
		}
	}
	assert.True(t, found1010)
	assert.True(t, found2010)
	assert.True(t, found4010)
}

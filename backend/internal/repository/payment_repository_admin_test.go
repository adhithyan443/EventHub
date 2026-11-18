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

func setupAdminPaymentTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	ddl := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		full_name VARCHAR(100) NOT NULL,
		email VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255) NOT NULL DEFAULT '',
		role VARCHAR(20) NOT NULL DEFAULT 'CUSTOMER',
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS organizers (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'APPROVED',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS organizer_profiles (
		id UUID PRIMARY KEY,
		organizer_id UUID NOT NULL,
		business_name VARCHAR(255) NOT NULL,
		business_type VARCHAR(100) NOT NULL DEFAULT 'INDIVIDUAL',
		phone VARCHAR(20) NOT NULL DEFAULT '',
		email VARCHAR(255) NOT NULL DEFAULT '',
		pan_number VARCHAR(50) NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_organizer_profiles_organizer_id ON organizer_profiles (organizer_id);

	CREATE TABLE IF NOT EXISTS events (
		id UUID PRIMARY KEY,
		organizer_id UUID NOT NULL,
		category_id UUID NOT NULL,
		event_type VARCHAR(20) NOT NULL DEFAULT 'IN_PERSON',
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status VARCHAR(30) NOT NULL DEFAULT 'PUBLISHED',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS bookings (
		id UUID PRIMARY KEY,
		booking_number VARCHAR(50) NOT NULL UNIQUE,
		user_id UUID NOT NULL,
		event_id UUID NOT NULL,
		subtotal NUMERIC(10,2) NOT NULL DEFAULT 0,
		platform_fee NUMERIC(10,2) NOT NULL DEFAULT 0,
		tax NUMERIC(10,2) NOT NULL DEFAULT 0,
		discount NUMERIC(10,2) NOT NULL DEFAULT 0,
		total NUMERIC(10,2) NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		booking_id UUID NOT NULL,
		transaction_id VARCHAR(100) UNIQUE,
		provider VARCHAR(30) NOT NULL DEFAULT 'RAZORPAY',
		provider_order_id VARCHAR(100) UNIQUE,
		payment_method VARCHAR(30) NOT NULL DEFAULT 'CARD',
		amount DECIMAL(10,2) NOT NULL,
		status VARCHAR(20) NOT NULL,
		paid_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	require.NoError(t, db.Exec(ddl).Error)
}

func setupAdminFinanceAccounts(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	acc1010 := uuid.New()
	acc2010 := uuid.New()
	acc4010 := uuid.New()
	acc5010 := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
		VALUES 
			(?, '1010', 'Razorpay Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
			(?, '2010', 'Organizer Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '4010', 'Platform Fee Revenue', 'REVENUE', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '5010', 'Payment Gateway Expense', 'EXPENSE', 'DEBIT', 'INR', TRUE, NOW())
	`, acc1010, acc2010, acc4010, acc5010).Error
	require.NoError(t, err)

	return acc1010, acc2010, acc4010, acc5010
}

func TestPaymentRepository_FindAdminPayments_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	// 1. Seed Common Hierarchy: User, Organizer, OrganizerProfile, Event, Bookings
	userID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, full_name, email)
		VALUES (?, 'Alice Customer', 'alice@example.com')
	`, userID).Error)

	orgUserID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, full_name, email)
		VALUES (?, 'Bob Organizer', 'bob@example.com')
	`, orgUserID).Error)

	orgID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO organizers (id, user_id)
		VALUES (?, ?)
	`, orgID, orgUserID).Error)

	orgProfID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO organizer_profiles (id, organizer_id, business_name)
		VALUES (?, ?, 'Awesome Events Corp')
	`, orgProfID, orgID).Error)

	eventID := uuid.New()
	categoryID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO events (id, organizer_id, category_id, title)
		VALUES (?, ?, ?, 'Mega Music Fest 2026')
	`, eventID, orgID, categoryID).Error)

	// Booking 1: Standard paid booking (subtotal 450 + fee 50 = total 500)
	bookingID := uuid.New()
	bookingNumber := "BK-2026-0001"
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, ?, ?, ?, 450.00, 50.00, 0.00, 0.00, 500.00, 'CONFIRMED')
	`, bookingID, bookingNumber, userID, eventID).Error)

	// Booking 2: Legitimate free booking (total 0)
	freeBookingID := uuid.New()
	freeBookingNumber := "BK-2026-FREE"
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, ?, ?, ?, 0.00, 0.00, 0.00, 0.00, 0.00, 'CONFIRMED')
	`, freeBookingID, freeBookingNumber, userID, eventID).Error)

	// Booking 3: Unconfirmed booking (status = PENDING)
	unconfirmedBookingID := uuid.New()
	unconfirmedBookingNumber := "BK-2026-UNCONF"
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, ?, ?, ?, 0.00, 0.00, 0.00, 0.00, 0.00, 'PENDING')
	`, unconfirmedBookingID, unconfirmedBookingNumber, userID, eventID).Error)

	// Booking 4: Booking with unsupported tax
	taxBookingID := uuid.New()
	taxBookingNumber := "BK-2026-TAX"
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, ?, ?, ?, 450.00, 50.00, 50.00, 0.00, 550.00, 'CONFIRMED')
	`, taxBookingID, taxBookingNumber, userID, eventID).Error)

	now := time.Now().UTC().Truncate(time.Second)
	tYesterday := now.Add(-24 * time.Hour)
	tTwoDaysAgo := now.Add(-48 * time.Hour)

	acc1010, acc2010, acc4010, acc5010 := setupAdminFinanceAccounts(t, db)

	// Case 1: Correct capture journal (ACCOUNTED)
	// Debit 1010 = 500, Credit 2010 = 450 (b.subtotal), Credit 4010 = 50 (b.platform_fee)
	payIDCorrect := uuid.New()
	txIDCorrect := "pay_tx_correct"
	orderIDCorrect := "order_correct"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'UPI', 500.00, 'SUCCESS', ?, ?, ?)
	`, payIDCorrect, bookingID, txIDCorrect, orderIDCorrect, now, now, now).Error)

	journalIDCorrect := uuid.New()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_correct', 'fp_correct', 'INR', 'Correct Payment', 'DRAFT', FALSE, ?)
		`, journalIDCorrect, payIDCorrect.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 500.00, 'INR', ?)
		`, uuid.New(), journalIDCorrect, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 450.00, 'INR', ?)
		`, uuid.New(), journalIDCorrect, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 50.00, 'INR', ?)
		`, uuid.New(), journalIDCorrect, acc4010, now).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
		`, now, journalIDCorrect).Error
	})
	require.NoError(t, err)

	// Case 2: Unbalanced capture journal -> UNALLOCATED
	// Debit 1010 = 500, Credit 2010 = 400 (debits 500 != credits 400)
	payIDUnbalanced := uuid.New()
	txIDUnbalanced := "pay_tx_unbalanced"
	orderIDUnbalanced := "order_unbalanced"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'UPI', 500.00, 'SUCCESS', ?, ?, ?)
	`, payIDUnbalanced, bookingID, txIDUnbalanced, orderIDUnbalanced, now, now, now).Error)

	require.NoError(t, db.Exec(`ALTER TABLE finance_journals DISABLE TRIGGER trg_validate_journal_double_entry`).Error)
	journalIDUnbalanced := uuid.New()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_unbalanced', 'fp_unbalanced', 'INR', 'Unbalanced Payment', 'DRAFT', FALSE, ?)
		`, journalIDUnbalanced, payIDUnbalanced.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 500.00, 'INR', ?)
		`, uuid.New(), journalIDUnbalanced, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 400.00, 'INR', ?)
		`, uuid.New(), journalIDUnbalanced, acc2010, now).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, journalIDUnbalanced).Error
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`ALTER TABLE finance_journals ENABLE TRIGGER trg_validate_journal_double_entry`).Error)

	// Case 3: Incorrect organizer payable amount -> UNALLOCATED
	// Debit 1010 = 500, Credit 2010 = 400 (expected 450), Credit 4010 = 100 (expected 50)
	payIDWrongOrgPayable := uuid.New()
	txIDWrongOrgPayable := "pay_tx_wrong_org"
	orderIDWrongOrgPayable := "order_wrong_org"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'UPI', 500.00, 'SUCCESS', ?, ?, ?)
	`, payIDWrongOrgPayable, bookingID, txIDWrongOrgPayable, orderIDWrongOrgPayable, now, now, now).Error)

	journalIDWrongOrg := uuid.New()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_wrong_org', 'fp_wrong_org', 'INR', 'Wrong Org Payment', 'DRAFT', FALSE, ?)
		`, journalIDWrongOrg, payIDWrongOrgPayable.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 500.00, 'INR', ?)
		`, uuid.New(), journalIDWrongOrg, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 400.00, 'INR', ?)
		`, uuid.New(), journalIDWrongOrg, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 100.00, 'INR', ?)
		`, uuid.New(), journalIDWrongOrg, acc4010, now).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
		`, now, journalIDWrongOrg).Error
	})
	require.NoError(t, err)

	// Case 4: Incorrect platform fee amount -> UNALLOCATED
	// Debit 1010 = 500, Credit 2010 = 450, Credit 4010 = 30 (expected 50), Credit 5010 = 20
	payIDWrongPlatformFee := uuid.New()
	txIDWrongPlatformFee := "pay_tx_wrong_fee"
	orderIDWrongPlatformFee := "order_wrong_fee"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'UPI', 500.00, 'SUCCESS', ?, ?, ?)
	`, payIDWrongPlatformFee, bookingID, txIDWrongPlatformFee, orderIDWrongPlatformFee, now, now, now).Error)

	journalIDWrongFee := uuid.New()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_wrong_fee', 'fp_wrong_fee', 'INR', 'Wrong Fee Payment', 'DRAFT', FALSE, ?)
		`, journalIDWrongFee, payIDWrongPlatformFee.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 500.00, 'INR', ?)
		`, uuid.New(), journalIDWrongFee, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 450.00, 'INR', ?)
		`, uuid.New(), journalIDWrongFee, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 30.00, 'INR', ?)
		`, uuid.New(), journalIDWrongFee, acc4010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 20.00, 'INR', ?)
		`, uuid.New(), journalIDWrongFee, acc5010, now).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
		`, now, journalIDWrongFee).Error
	})
	require.NoError(t, err)

	// Case 5: Legitimate Free booking payment (amount = 0, status = SUCCESS, booking confirmed, booking total = 0) -> ACCOUNTED
	payIDZeroFree := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, NULL, 'CARD', 0.00, 'SUCCESS', NULL, ?, ?)
	`, payIDZeroFree, freeBookingID, tYesterday, tYesterday).Error)

	// Case 6: Captured payment with unsupported tax (REFUND_REQUIRED) -> UNALLOCATED
	payIDTaxUnsupported := uuid.New()
	txIDTax := "pay_tx_tax"
	orderIDTax := "order_tax"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'CARD', 550.00, 'REFUND_REQUIRED', ?, ?, ?)
	`, payIDTaxUnsupported, taxBookingID, txIDTax, orderIDTax, now, now, now).Error)

	// Case 7: Captured payment with reversed journal -> UNALLOCATED
	payIDReversed := uuid.New()
	txIDReversed := "pay_tx_reversed"
	orderIDReversed := "order_reversed"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'NET_BANKING', 750.00, 'SUCCESS', ?, ?, ?)
	`, payIDReversed, bookingID, txIDReversed, orderIDReversed, now, now, now).Error)

	journalIDReversed := uuid.New()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_rev', 'fp_rev', 'INR', 'Payment to be reversed', 'DRAFT', FALSE, ?)
		`, journalIDReversed, payIDReversed.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 750.00, 'INR', ?)
		`, uuid.New(), journalIDReversed, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 750.00, 'INR', ?)
		`, uuid.New(), journalIDReversed, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
		`, now, journalIDReversed).Error; err != nil {
			return err
		}

		revJournalID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, reversal_reference_id, created_at)
			VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', ?, 'key_rev_child', 'fp_rev_child', 'INR', 'Reversal Child', 'DRAFT', FALSE, ?, ?)
		`, revJournalID, payIDReversed.String(), journalIDReversed, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'CREDIT', 750.00, 'INR', ?)
		`, uuid.New(), revJournalID, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', 750.00, 'INR', ?)
		`, uuid.New(), revJournalID, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?
		`, now, revJournalID).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = ? WHERE id = ?
		`, revJournalID, now, journalIDReversed).Error
	})
	require.NoError(t, err)

	// Case 8: Pending payment attempt (no gateway capture) -> NOT_APPLICABLE
	payIDPending := uuid.New()
	orderIDPending := "order_pending"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, ?, 'UPI', 200.00, 'PENDING', NULL, ?, ?)
	`, payIDPending, bookingID, orderIDPending, tTwoDaysAgo, tTwoDaysAgo).Error)

	// Case 9: Failed payment attempt (no gateway capture) -> NOT_APPLICABLE
	payIDFailed := uuid.New()
	orderIDFailed := "order_failed"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, ?, 'WALLET', 150.00, 'FAILED', NULL, ?, ?)
	`, payIDFailed, bookingID, orderIDFailed, tTwoDaysAgo, tTwoDaysAgo).Error)

	// Case 10: Zero-value payment on positive-total booking (mismatch) -> NOT_APPLICABLE
	payIDZeroMismatch := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, NULL, 'CARD', 0.00, 'SUCCESS', NULL, ?, ?)
	`, payIDZeroMismatch, bookingID, tTwoDaysAgo, tTwoDaysAgo).Error)

	// Case 11: Zero-value payment on unconfirmed booking (status = PENDING) -> NOT_APPLICABLE
	payIDZeroUnconfirmed := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, NULL, 'CARD', 0.00, 'SUCCESS', NULL, ?, ?)
	`, payIDZeroUnconfirmed, unconfirmedBookingID, tTwoDaysAgo, tTwoDaysAgo).Error)

	t.Run("lists all payments with full relationship and allocation fields", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Page:  1,
			Limit: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, int64(11), res.TotalRecords)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.Limit)
		assert.Equal(t, 1, res.TotalPages)
		assert.Len(t, res.Payments, 11)

		payMap := make(map[string]domain.AdminPaymentListItem)
		for _, p := range res.Payments {
			payMap[p.ID] = p
		}

		// Verify Correct capture journal -> ACCOUNTED
		pCorrect, exists := payMap[payIDCorrect.String()]
		require.True(t, exists)
		assert.Equal(t, bookingNumber, pCorrect.BookingNumber)
		assert.Equal(t, "Alice Customer", pCorrect.Customer.Name)
		assert.Equal(t, "alice@example.com", pCorrect.Customer.Email)
		assert.Equal(t, "Mega Music Fest 2026", pCorrect.Event.Title)
		assert.Equal(t, "Awesome Events Corp", pCorrect.Organizer.BusinessName)
		assert.True(t, decimal.NewFromInt(500).Equal(pCorrect.Amount))
		assert.Equal(t, "INR", pCorrect.Currency)
		assert.Equal(t, domain.PaymentStatusSuccess, pCorrect.Status)
		assert.True(t, pCorrect.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusAccounted, pCorrect.AllocationStatus)

		// Verify Legitimate Free booking -> ACCOUNTED
		pFree, exists := payMap[payIDZeroFree.String()]
		require.True(t, exists)
		assert.Equal(t, freeBookingNumber, pFree.BookingNumber)
		assert.True(t, decimal.Zero.Equal(pFree.Amount))
		assert.Equal(t, domain.PaymentStatusSuccess, pFree.Status)
		assert.False(t, pFree.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusAccounted, pFree.AllocationStatus)

		// Verify Unbalanced journal -> UNALLOCATED
		pUnbalanced, exists := payMap[payIDUnbalanced.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pUnbalanced.Status)
		assert.True(t, pUnbalanced.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, pUnbalanced.AllocationStatus)

		// Verify Incorrect organizer payable amount -> UNALLOCATED
		pWrongOrg, exists := payMap[payIDWrongOrgPayable.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pWrongOrg.Status)
		assert.True(t, pWrongOrg.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, pWrongOrg.AllocationStatus)

		// Verify Incorrect platform fee amount -> UNALLOCATED
		pWrongFee, exists := payMap[payIDWrongPlatformFee.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pWrongFee.Status)
		assert.True(t, pWrongFee.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, pWrongFee.AllocationStatus)

		// Verify Unsupported tax -> UNALLOCATED
		pTax, exists := payMap[payIDTaxUnsupported.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusRefundRequired, pTax.Status)
		assert.True(t, pTax.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, pTax.AllocationStatus)

		// Verify Reversed journal -> UNALLOCATED
		pRev, exists := payMap[payIDReversed.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pRev.Status)
		assert.True(t, pRev.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, pRev.AllocationStatus)

		// Verify Pending -> NOT_APPLICABLE
		pPending, exists := payMap[payIDPending.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusPending, pPending.Status)
		assert.False(t, pPending.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusNotApplicable, pPending.AllocationStatus)

		// Verify Failed -> NOT_APPLICABLE
		pFailed, exists := payMap[payIDFailed.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusFailed, pFailed.Status)
		assert.False(t, pFailed.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusNotApplicable, pFailed.AllocationStatus)

		// Verify Zero mismatch -> NOT_APPLICABLE
		pMismatch, exists := payMap[payIDZeroMismatch.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pMismatch.Status)
		assert.False(t, pMismatch.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusNotApplicable, pMismatch.AllocationStatus)

		// Verify Zero unconfirmed -> NOT_APPLICABLE
		pUnconf, exists := payMap[payIDZeroUnconfirmed.String()]
		require.True(t, exists)
		assert.Equal(t, domain.PaymentStatusSuccess, pUnconf.Status)
		assert.False(t, pUnconf.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusNotApplicable, pUnconf.AllocationStatus)
	})

	t.Run("filters by allocation status ACCOUNTED", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			AllocationStatus: "ACCOUNTED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), res.TotalRecords) // payIDCorrect, payIDZeroFree
		assert.Len(t, res.Payments, 2)
		for _, p := range res.Payments {
			assert.Equal(t, domain.AllocationStatusAccounted, p.AllocationStatus)
		}
	})

	t.Run("filters by allocation status UNALLOCATED", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			AllocationStatus: "UNALLOCATED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), res.TotalRecords) // Unbalanced, WrongOrg, WrongFee, Tax, Reversed
		assert.Len(t, res.Payments, 5)
		for _, p := range res.Payments {
			assert.Equal(t, domain.AllocationStatusUnallocated, p.AllocationStatus)
		}
	})

	t.Run("filters by allocation status NOT_APPLICABLE", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			AllocationStatus: "NOT_APPLICABLE",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), res.TotalRecords) // Pending, Failed, Mismatch, Unconfirmed
		assert.Len(t, res.Payments, 4)
		for _, p := range res.Payments {
			assert.Equal(t, domain.AllocationStatusNotApplicable, p.AllocationStatus)
		}
	})

	t.Run("filters by payment status SUCCESS", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Status: "SUCCESS",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(8), res.TotalRecords)
		for _, p := range res.Payments {
			assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
		}
	})

	t.Run("filters by payment status REFUND_REQUIRED", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Status: "REFUND_REQUIRED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Equal(t, payIDTaxUnsupported.String(), res.Payments[0].ID)
	})

	t.Run("searches by transaction ID", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Search: txIDCorrect,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Equal(t, payIDCorrect.String(), res.Payments[0].ID)
	})

	t.Run("searches by booking number", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Search: freeBookingNumber,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Equal(t, payIDZeroFree.String(), res.Payments[0].ID)
	})

	t.Run("searches by customer email", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Search: "alice@example.com",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(11), res.TotalRecords)
	})

	t.Run("searches by customer name", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Search: "Alice Customer",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(11), res.TotalRecords)
	})

	t.Run("filters by date range", func(t *testing.T) {
		from := now.Add(-12 * time.Hour)
		to := now.Add(12 * time.Hour)
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			From: &from,
			To:   &to,
		})
		require.NoError(t, err)
		// PayCorrect, PayUnbalanced, PayWrongOrg, PayWrongFee, PayTax, PayReversed were created at 'now'
		assert.Equal(t, int64(6), res.TotalRecords)
	})

	t.Run("paginates results accurately", func(t *testing.T) {
		p1, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Page:  1,
			Limit: 3,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(11), p1.TotalRecords)
		assert.Equal(t, 4, p1.TotalPages)
		assert.Len(t, p1.Payments, 3)

		p2, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Page:  2,
			Limit: 3,
		})
		require.NoError(t, err)
		assert.Len(t, p2.Payments, 3)
		assert.NotEqual(t, p1.Payments[0].ID, p2.Payments[0].ID)
	})

	t.Run("protects against integer overflow and handles large page safely", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Page:  150_000,
			Limit: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, int64(11), res.TotalRecords)
		assert.Equal(t, 100_000, res.Page) // Clamped to maxSafePage
		assert.Equal(t, 0, len(res.Payments))
	})

	t.Run("organizer profile join uniqueness prevents row duplication", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			Limit: 100,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(11), res.TotalRecords)
		assert.Len(t, res.Payments, 11)
	})
}

func TestPaymentRepository_TimestampSemantics(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	var sessionTZ string
	err := db.Raw("SHOW TIMEZONE").Scan(&sessionTZ).Error
	require.NoError(t, err)
	t.Logf("PostgreSQL session timezone: %s", sessionTZ)

	// Create a user and booking to attach payment
	uID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'TS User', 'ts@example.com')`, uID).Error)
	eID := uuid.New()
	orgID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID, uID).Error)
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'TS Event')`, eID, orgID, uuid.New()).Error)
	bID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, total, status) VALUES (?, 'BK-TS-1', ?, ?, 100, 'CONFIRMED')`, bID, uID, eID).Error)

	// 1. Test Payment Creation timestamp semantics
	pay := &domain.Payment{
		BookingID: bID.String(),
		Provider:  domain.PaymentProviderRazorpay,
		Method:    domain.PaymentMethodUPI,
		Amount:    decimal.NewFromInt(100),
		Status:    domain.PaymentStatusPending,
	}
	require.NoError(t, repo.Create(ctx, pay))

	var rawCreatedAt string
	require.NoError(t, db.Raw(`SELECT to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') FROM payments WHERE id = ?`, pay.ID).Scan(&rawCreatedAt).Error)
	t.Logf("Raw DB created_at (string): %s", rawCreatedAt)
	t.Logf("Go model created_at: %s (Location: %s)", pay.CreatedAt.String(), pay.CreatedAt.Location().String())

	// 2. Test MarkSuccessful timestamp semantics with non-UTC (local) time
	localTime := time.Now()
	txID := "tx_ts_test"
	require.NoError(t, repo.MarkSuccessful(ctx, pay.ID, txID, localTime))

	var rawPaidAt string
	require.NoError(t, db.Raw(`SELECT to_char(paid_at, 'YYYY-MM-DD HH24:MI:SS') FROM payments WHERE id = ?`, pay.ID).Scan(&rawPaidAt).Error)
	t.Logf("Raw DB paid_at (string): %s", rawPaidAt)
	t.Logf("Input local paidAt: %s (Location: %s)", localTime.String(), localTime.Location().String())
	t.Logf("Input UTC paidAt: %s", localTime.UTC().Format("2006-01-02 15:04:05"))

	// 3. Read back via FindByID
	found, err := repo.FindByID(ctx, pay.ID)
	require.NoError(t, err)
	require.NotNil(t, found.PaidAt)
	t.Logf("FindByID read PaidAt: %s (Location: %s)", found.PaidAt.String(), found.PaidAt.Location().String())
	t.Logf("FindByID read CreatedAt: %s (Location: %s)", found.CreatedAt.String(), found.CreatedAt.Location().String())

	// 4. Test FindAdminPayments date filtering with UTC vs Local bounds
	dayStartUTC := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 0, 0, 0, 0, time.UTC)
	dayEndUTC := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 23, 59, 59, 999999999, time.UTC)
	resUTC, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
		Search: pay.ID,
		From:   &dayStartUTC,
		To:     &dayEndUTC,
	})
	require.NoError(t, err)
	t.Logf("FindAdminPayments with UTC day bounds returned %d records", resUTC.TotalRecords)
	assert.Equal(t, int64(1), resUTC.TotalRecords)

	// Compare with raw SQL comparison
	var countMatches int64
	err = db.Raw(`SELECT COUNT(*) FROM payments WHERE id = ? AND COALESCE(paid_at, created_at) >= ? AND COALESCE(paid_at, created_at) <= ?`,
		pay.ID, dayStartUTC, dayEndUTC).Scan(&countMatches).Error
	require.NoError(t, err)
	t.Logf("Raw SQL comparison with day bounds returned: %d", countMatches)
	assert.Equal(t, int64(1), countMatches)
}

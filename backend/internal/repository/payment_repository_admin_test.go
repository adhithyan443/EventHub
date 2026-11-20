package repository_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	CREATE TABLE IF NOT EXISTS booking_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		booking_id UUID NOT NULL,
		seat_id UUID,
		ticket_type_id UUID,
		quantity INTEGER NOT NULL,
		unit_price NUMERIC(10, 2) NOT NULL,
		total_price NUMERIC(10, 2) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_booking_items_booking_id ON booking_items(booking_id);

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

	CREATE TABLE IF NOT EXISTS refunds (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		booking_id UUID NOT NULL,
		payment_id UUID NOT NULL,
		gateway_refund_id VARCHAR(100),
		amount NUMERIC(10, 2) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		reason VARCHAR(255) NOT NULL DEFAULT 'CUSTOMER_CANCELLATION',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		processed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_refunds_booking_id ON refunds(booking_id);
	CREATE INDEX IF NOT EXISTS idx_refunds_payment_id ON refunds(payment_id);
	`

	require.NoError(t, db.Exec(ddl).Error)

	truncateSQL := `
		TRUNCATE TABLE 
			refunds, 
			payments, 
			bookings, 
			events, 
			organizer_profiles, 
			organizers, 
			users, 
			finance_ledger_entries, 
			finance_journals, 
			finance_accounts 
		CASCADE;
	`
	require.NoError(t, db.Exec(truncateSQL).Error)
}

func setupAdminFinanceAccounts(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	acc1010, acc2010, _, acc4010, acc5010 := setupAdminFinanceAccountsWithTax(t, db)
	return acc1010, acc2010, acc4010, acc5010
}

func setupAdminFinanceAccountsWithTax(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	acc1010 := uuid.New()
	acc2010 := uuid.New()
	acc2030 := uuid.New()
	acc4010 := uuid.New()
	acc5010 := uuid.New()

	err := db.Exec(`
		INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
		VALUES 
			(?, '1010', 'Razorpay Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
			(?, '2010', 'Organizer Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '2030', 'Tax Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '4010', 'Platform Fee Revenue', 'REVENUE', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '5010', 'Payment Gateway Expense', 'EXPENSE', 'DEBIT', 'INR', TRUE, NOW())
	`, acc1010, acc2010, acc2030, acc4010, acc5010).Error
	require.NoError(t, err)

	return acc1010, acc2010, acc2030, acc4010, acc5010
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

	// Case 6: Captured payment with no journal recorded (e.g. reservation expired/refund required) -> UNALLOCATED
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

		// Verify Captured payment with no journal -> UNALLOCATED
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

	t.Run("filters by event ID", func(t *testing.T) {
		res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			EventID: &eventID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(11), res.TotalRecords)

		otherEventID := uuid.New()
		resOther, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			EventID: &otherEventID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), resOther.TotalRecords)
		assert.Empty(t, resOther.Payments)
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

func TestPaymentRepository_GetEventPaymentSummaries_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, acc2010, acc4010, _ := setupAdminFinanceAccounts(t, db)

	// 1. Seed Organizers and Profiles
	// Org 1: Rockstar Productions
	orgUser1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org One', 'org1@example.com')`, orgUser1).Error)
	orgID1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID1, orgUser1).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Rockstar Productions')`, uuid.New(), orgID1).Error)

	// Org 2: Zenith Events
	orgUser2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org Two', 'org2@example.com')`, orgUser2).Error)
	orgID2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID2, orgUser2).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Zenith Events')`, uuid.New(), orgID2).Error)

	// Customer User
	customerID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Dave Customer', 'dave@example.com')`, customerID).Error)

	now := time.Now().UTC().Truncate(time.Second)
	tYesterday := now.Add(-24 * time.Hour)

	// 2. Seed 5 Events
	eventID1 := uuid.New() // Rock Anthem 2026
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, created_at) VALUES (?, ?, ?, 'Rock Anthem 2026', ?)`, eventID1, orgID1, uuid.New(), now).Error)

	eventID2 := uuid.New() // Jazz Night Under The Stars
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, created_at) VALUES (?, ?, ?, 'Jazz Night Under The Stars', ?)`, eventID2, orgID1, uuid.New(), now).Error)

	eventID3 := uuid.New() // Electro Wave Festival
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, created_at) VALUES (?, ?, ?, 'Electro Wave Festival', ?)`, eventID3, orgID2, uuid.New(), now).Error)

	eventID4 := uuid.New() // Classical Symphony Showcase
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, created_at) VALUES (?, ?, ?, 'Classical Symphony Showcase', ?)`, eventID4, orgID2, uuid.New(), tYesterday).Error)

	eventID5 := uuid.New() // Tech Summit Global
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, created_at) VALUES (?, ?, ?, 'Tech Summit Global', ?)`, eventID5, orgID2, uuid.New(), tYesterday).Error)

	// --- Event 1 Payments & Refunds ---
	// Scenario 1: Multiple successful partial refunds against one payment (Amount: 1000, 2x300 refunds)
	b1A := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E1-A', ?, ?, 900.00, 100.00, 1000.00, 'CONFIRMED', ?)`, b1A, customerID, eventID1, now).Error)
	p1_1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e1_1', 'order_e1_1', 'UPI', 1000.00, 'REFUND_REQUIRED', ?, ?)`, p1_1, b1A, now, now).Error)
	rf1_1 := "rf_e1_1"
	require.NoError(t, db.Exec(`INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, processed_at, created_at) VALUES (?, ?, ?, ?, 300.00, 'SUCCESS', ?, ?)`, uuid.New(), b1A, p1_1, rf1_1, now, now).Error)
	rf1_2 := "rf_e1_2"
	require.NoError(t, db.Exec(`INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, processed_at, created_at) VALUES (?, ?, ?, ?, 300.00, 'SUCCESS', ?, ?)`, uuid.New(), b1A, p1_1, rf1_2, now, now).Error)

	// Scenario 2: Successful and failed refunds associated with same payment (Amount: 500, 1x500 SUCCESS, 1x500 FAILED)
	b1B := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E1-B', ?, ?, 450.00, 50.00, 500.00, 'CONFIRMED', ?)`, b1B, customerID, eventID1, now).Error)
	p1_2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e1_2', 'order_e1_2', 'CARD', 500.00, 'REFUND_REQUIRED', ?, ?)`, p1_2, b1B, now, now).Error)
	rf2_succ := "rf_e1_succ"
	require.NoError(t, db.Exec(`INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, processed_at, created_at) VALUES (?, ?, ?, ?, 500.00, 'SUCCESS', ?, ?)`, uuid.New(), b1B, p1_2, rf2_succ, now, now).Error)
	rf2_fail := "rf_e1_fail"
	require.NoError(t, db.Exec(`INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, processed_at, created_at) VALUES (?, ?, ?, ?, 500.00, 'FAILED', ?, ?)`, uuid.New(), b1B, p1_2, rf2_fail, now, now).Error)

	// Scenario 3: REFUND_REQUIRED payment without refund record (Amount: 400, 0 refunds)
	b1C := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E1-C', ?, ?, 360.00, 40.00, 400.00, 'CONFIRMED', ?)`, b1C, customerID, eventID1, now).Error)
	p1_3 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e1_3', 'order_e1_3', 'NET_BANKING', 400.00, 'REFUND_REQUIRED', ?, ?)`, p1_3, b1C, now, now).Error)

	// --- Event 2 Payments & Journals ---
	// Scenario 4: Captured payment with valid posted journal (ACCOUNTED)
	b2A := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E2-A', ?, ?, 450.00, 50.00, 500.00, 'CONFIRMED', ?)`, b2A, customerID, eventID2, now).Error)
	p2_1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e2_1', 'order_e2_1', 'UPI', 500.00, 'SUCCESS', ?, ?)`, p2_1, b2A, now, now).Error)

	jID_e2 := uuid.New()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_e2_1', 'fp_e2_1', 'INR', 'Accounted E2 Payment', 'DRAFT', FALSE, ?)
		`, jID_e2, p2_1.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'DEBIT', 500.00, 'INR', ?)`, uuid.New(), jID_e2, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'CREDIT', 450.00, 'INR', ?)`, uuid.New(), jID_e2, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'CREDIT', 50.00, 'INR', ?)`, uuid.New(), jID_e2, acc4010, now).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, jID_e2).Error
	})
	require.NoError(t, err)

	// Scenario 5: Captured payment with incomplete or reversed journal (UNALLOCATED)
	b2B := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E2-B', ?, ?, 600.00, 0.00, 600.00, 'CONFIRMED', ?)`, b2B, customerID, eventID2, now).Error)
	p2_2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e2_2', 'order_e2_2', 'CARD', 600.00, 'SUCCESS', ?, ?)`, p2_2, b2B, now, now).Error)
	// Journal is reversed:
	jID_e2_rev := uuid.New()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
			VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_e2_2', 'fp_e2_2', 'INR', 'Reversed E2 Payment', 'DRAFT', FALSE, ?)
		`, jID_e2_rev, p2_2.String(), now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'DEBIT', 600.00, 'INR', ?)`, uuid.New(), jID_e2_rev, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'CREDIT', 600.00, 'INR', ?)`, uuid.New(), jID_e2_rev, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, jID_e2_rev).Error; err != nil {
			return err
		}
		revJournalID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, reversal_reference_id, created_at)
			VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', ?, 'key_e2_rev_child', 'fp_e2_rev_child', 'INR', 'Reversal Child', 'DRAFT', FALSE, ?, ?)
		`, revJournalID, p2_2.String(), jID_e2_rev, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'CREDIT', 600.00, 'INR', ?)`, uuid.New(), revJournalID, acc1010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at) VALUES (?, ?, ?, 'DEBIT', 600.00, 'INR', ?)`, uuid.New(), revJournalID, acc2010, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, revJournalID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = ? WHERE id = ?`, revJournalID, now, jID_e2_rev).Error
	})
	require.NoError(t, err)

	// --- Event 3 Payments ---
	// Scenario 6: Multiple payment attempts for one booking (FAILED, PENDING, SUCCESS)
	b3A := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E3-A', ?, ?, 200.00, 20.00, 220.00, 'CONFIRMED', ?)`, b3A, customerID, eventID3, now).Error)
	// Attempt 1: FAILED
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, NULL, 'order_e3_fail', 'UPI', 220.00, 'FAILED', NULL, ?)`, uuid.New(), b3A, now).Error)
	// Attempt 2: PENDING
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, NULL, 'order_e3_pend', 'UPI', 220.00, 'PENDING', NULL, ?)`, uuid.New(), b3A, now).Error)
	// Attempt 3: SUCCESS (Captured)
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e3_succ', 'order_e3_succ', 'UPI', 220.00, 'SUCCESS', ?, ?)`, uuid.New(), b3A, now, now).Error)

	// --- Event 4 Payments ---
	b4A := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E4-A', ?, ?, 150.00, 0.00, 150.00, 'CONFIRMED', ?)`, b4A, customerID, eventID4, tYesterday).Error)
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e4_1', 'order_e4_1', 'CARD', 150.00, 'SUCCESS', ?, ?)`, uuid.New(), b4A, tYesterday, tYesterday).Error)

	// --- Event 5 Payments ---
	b5A := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status, created_at) VALUES (?, 'BK-E5-A', ?, ?, 350.00, 0.00, 350.00, 'CONFIRMED', ?)`, b5A, customerID, eventID5, tYesterday).Error)
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_e5_1', 'order_e5_1', 'CARD', 350.00, 'SUCCESS', ?, ?)`, uuid.New(), b5A, tYesterday, tYesterday).Error)

	t.Run("Scenario 1: multiple successful partial refunds against one payment (fan-out protection)", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID1,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Summaries, 1)

		e1 := res.Summaries[0]
		assert.Equal(t, eventID1, e1.EventID)
		assert.Equal(t, "Rock Anthem 2026", e1.EventTitle)
		assert.Equal(t, "Rockstar Productions", e1.OrganizerBusinessName)
		assert.Equal(t, int64(3), e1.TotalPaymentsCount) // Exactly 3 payments, NOT multiplied by refunds!
		assert.Equal(t, int64(0), e1.SuccessfullyCompletedPaymentsCount)
		assert.True(t, decimal.NewFromInt(1900).Equal(e1.GrossCapturedAmount))
		assert.Equal(t, int64(2), e1.RefundRequiredCount)               // P1_1 (400 outstanding) + P1_3 (400 outstanding)
		assert.Equal(t, int64(2), e1.PaymentsWithSuccessfulRefundsCount) // P1_1 and P1_2 have successful refunds
	})

	t.Run("Scenario 2: successful and failed refunds associated with same payment", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID1,
		})
		require.NoError(t, err)
		require.Len(t, res.Summaries, 1)
		// P1_2 had 500 successful refund and 500 failed refund.
		// Since 500 is fully refunded, it is NOT in RefundRequiredCount.
		// Thus RefundRequiredCount is 2 (P1_1 and P1_3 only), not 3.
		assert.Equal(t, int64(2), res.Summaries[0].RefundRequiredCount)
	})

	t.Run("Scenario 3: REFUND_REQUIRED payment without refund record", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID1,
		})
		require.NoError(t, err)
		require.Len(t, res.Summaries, 1)
		// P1_3 has no refund record, so its outstanding refund is 400 > 0, counted in RefundRequiredCount.
		assert.Equal(t, int64(2), res.Summaries[0].RefundRequiredCount)
	})

	t.Run("Scenario 4: captured payment with valid posted journal (ACCOUNTED)", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID2,
		})
		require.NoError(t, err)
		require.Len(t, res.Summaries, 1)

		e2 := res.Summaries[0]
		assert.Equal(t, int64(2), e2.TotalPaymentsCount)
		assert.Equal(t, int64(2), e2.SuccessfullyCompletedPaymentsCount)
		assert.True(t, decimal.NewFromInt(1100).Equal(e2.GrossCapturedAmount))
		// P2_1 is ACCOUNTED, P2_2 is UNALLOCATED, so UnallocatedCapturedCount is exactly 1.
		assert.Equal(t, int64(1), e2.UnallocatedCapturedCount)
	})

	t.Run("Scenario 5: captured payment with incomplete or reversed journal (UNALLOCATED)", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID2,
		})
		require.NoError(t, err)
		require.Len(t, res.Summaries, 1)
		// P2_2 has reversed journal, which is not valid posted non-reversed, so it's UNALLOCATED.
		assert.Equal(t, int64(1), res.Summaries[0].UnallocatedCapturedCount)
	})

	t.Run("Scenario 6: multiple payment attempts for one booking", func(t *testing.T) {
		res, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			EventID: &eventID3,
		})
		require.NoError(t, err)
		require.Len(t, res.Summaries, 1)

		e3 := res.Summaries[0]
		assert.Equal(t, int64(3), e3.TotalPaymentsCount)                          // FAILED, PENDING, SUCCESS
		assert.Equal(t, int64(1), e3.SuccessfullyCompletedPaymentsCount)         // Only SUCCESS has status = SUCCESS and confirmed booking
		assert.Equal(t, int64(1), e3.PendingPaymentsCount)                       // PENDING
		assert.Equal(t, int64(1), e3.FailedPaymentsCount)                        // FAILED
		assert.True(t, decimal.NewFromInt(220).Equal(e3.GrossCapturedAmount))    // Does NOT sum failed or pending!
		assert.Equal(t, int64(1), e3.UnallocatedCapturedCount)                  // Captured payment without journal is UNALLOCATED
	})

	t.Run("Scenario 7: overall summary consistency across pagination pages", func(t *testing.T) {
		expectedGross := decimal.NewFromInt(3720) // 1900 + 1100 + 220 + 150 + 350
		expectedOverall := domain.AdminOverallPaymentSummary{
			TotalEvents:              5,
			GrossCapturedAmount:      expectedGross,
			Currency:                 finance.CurrencyINR,
			RefundRequiredCount:      2,
			UnallocatedCapturedCount: 7, // E1(3) + E2(1) + E3(1) + E4(1) + E5(1)
		}

		// Page 1 (limit 2)
		p1, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			Page:  1,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), p1.TotalRecords)
		assert.Equal(t, 1, p1.Page)
		assert.Equal(t, 2, p1.Limit)
		assert.Equal(t, 3, p1.TotalPages)
		assert.Len(t, p1.Summaries, 2)
		assert.Equal(t, expectedOverall.TotalEvents, p1.OverallSummary.TotalEvents)
		assert.True(t, expectedOverall.GrossCapturedAmount.Equal(p1.OverallSummary.GrossCapturedAmount))
		assert.Equal(t, expectedOverall.RefundRequiredCount, p1.OverallSummary.RefundRequiredCount)
		assert.Equal(t, expectedOverall.UnallocatedCapturedCount, p1.OverallSummary.UnallocatedCapturedCount)

		// Check sorting: E1 (1900) should be first, E2 (1100) second
		assert.Equal(t, eventID1, p1.Summaries[0].EventID)
		assert.Equal(t, eventID2, p1.Summaries[1].EventID)

		// Page 2 (limit 2)
		p2, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			Page:  2,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), p2.TotalRecords)
		assert.Equal(t, 2, p2.Page)
		assert.Len(t, p2.Summaries, 2)
		// Check sorting: E5 (350) should be third, E3 (220) fourth
		assert.Equal(t, eventID5, p2.Summaries[0].EventID)
		assert.Equal(t, eventID3, p2.Summaries[1].EventID)
		// Overall summary must be IDENTICAL across pages!
		assert.Equal(t, p1.OverallSummary.TotalEvents, p2.OverallSummary.TotalEvents)
		assert.True(t, p1.OverallSummary.GrossCapturedAmount.Equal(p2.OverallSummary.GrossCapturedAmount))
		assert.Equal(t, p1.OverallSummary.RefundRequiredCount, p2.OverallSummary.RefundRequiredCount)
		assert.Equal(t, p1.OverallSummary.UnallocatedCapturedCount, p2.OverallSummary.UnallocatedCapturedCount)

		// Page 3 (limit 2)
		p3, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			Page:  3,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), p3.TotalRecords)
		assert.Equal(t, 3, p3.Page)
		assert.Len(t, p3.Summaries, 1)
		// Fifth: E4 (150)
		assert.Equal(t, eventID4, p3.Summaries[0].EventID)
		// Overall summary must still be IDENTICAL!
		assert.Equal(t, p1.OverallSummary.TotalEvents, p3.OverallSummary.TotalEvents)
		assert.True(t, p1.OverallSummary.GrossCapturedAmount.Equal(p3.OverallSummary.GrossCapturedAmount))
		assert.Equal(t, p1.OverallSummary.RefundRequiredCount, p3.OverallSummary.RefundRequiredCount)
		assert.Equal(t, p1.OverallSummary.UnallocatedCapturedCount, p3.OverallSummary.UnallocatedCapturedCount)
	})

	t.Run("Scenario 8: search and date-filter consistency affecting both event summaries and overall summary", func(t *testing.T) {
		// 8a. Search by Event Title
		resSearchTitle, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			Search: "Rock Anthem",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resSearchTitle.TotalRecords)
		assert.Len(t, resSearchTitle.Summaries, 1)
		assert.Equal(t, eventID1, resSearchTitle.Summaries[0].EventID)
		assert.Equal(t, int64(1), resSearchTitle.OverallSummary.TotalEvents)
		assert.True(t, decimal.NewFromInt(1900).Equal(resSearchTitle.OverallSummary.GrossCapturedAmount))
		assert.Equal(t, int64(2), resSearchTitle.OverallSummary.RefundRequiredCount)

		// 8b. Search by Organizer Business Name
		resSearchOrg, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			Search: "Zenith Events",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), resSearchOrg.TotalRecords) // Events 3, 4, 5
		assert.Len(t, resSearchOrg.Summaries, 3)
		assert.Equal(t, int64(3), resSearchOrg.OverallSummary.TotalEvents)
		expectedZenithGross := decimal.NewFromInt(220 + 150 + 350)
		assert.True(t, expectedZenithGross.Equal(resSearchOrg.OverallSummary.GrossCapturedAmount))

		// 8c. Date range filter: only payments from "now - 1h" to "now + 1h"
		fromWindow := now.Add(-1 * time.Hour)
		toWindow := now.Add(1 * time.Hour)
		resDate, err := repo.GetEventPaymentSummaries(ctx, domain.AdminPaymentFilter{
			From: &fromWindow,
			To:   &toWindow,
		})
		require.NoError(t, err)
		// Events 1, 2, 3 have payments in this window. Events 4 & 5 (tYesterday) are excluded.
		assert.Equal(t, int64(3), resDate.TotalRecords)
		assert.Len(t, resDate.Summaries, 3)
		assert.Equal(t, int64(3), resDate.OverallSummary.TotalEvents)
		expectedWindowGross := decimal.NewFromInt(1900 + 1100 + 220)
		assert.True(t, expectedWindowGross.Equal(resDate.OverallSummary.GrossCapturedAmount))
	})
}

func TestPaymentRepository_Postgres_TimezoneInvariance(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	ctx := context.Background()

	// Seed Hierarchy
	uID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'TZ User', 'tz@example.com')`, uID).Error)
	orgID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID, uID).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'TZ Org Corp')`, uuid.New(), orgID).Error)
	eventID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Timezone Invariance Showcase')`, eventID, orgID, uuid.New()).Error)

	// Booking 1: Subtotal 500
	bID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, total, status) VALUES (?, 'BK-TZ-1', ?, ?, 450.00, 50.00, 500.00, 'CONFIRMED')`, bID, uID, eventID).Error)

	// Seed payments at specific UTC timestamps
	// P1: 2026-08-15 10:00:00 UTC (In Asia/Kolkata +05:30: 2026-08-15 15:30:00)
	tP1 := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	p1ID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_tz_1', 'order_tz_1', 'UPI', 500.00, 'SUCCESS', ?, ?)`, p1ID, bID, tP1, tP1).Error)

	// P2: 2026-08-15 20:00:00 UTC (In Asia/Kolkata +05:30: 2026-08-16 01:30:00 NEXT DAY!)
	tP2 := time.Date(2026, 8, 15, 20, 0, 0, 0, time.UTC)
	p2ID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_tz_2', 'order_tz_2', 'CARD', 500.00, 'SUCCESS', ?, ?)`, p2ID, bID, tP2, tP2).Error)

	// P3: 2026-08-16 12:00:00 UTC (In Asia/Kolkata +05:30: 2026-08-16 17:30:00)
	tP3 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	p3ID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at) VALUES (?, ?, 'tx_tz_3', 'order_tz_3', 'NET_BANKING', 500.00, 'SUCCESS', ?, ?)`, p3ID, bID, tP3, tP3).Error)

	// Pin to a single dedicated PostgreSQL connection
	sqlDB, err := db.DB()
	require.NoError(t, err)

	conn, err := sqlDB.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	pinnedDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: conn,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	repo := repository.NewPaymentRepository(pinnedDB, slog.Default())

	// Date filter covering 2026-08-15 (entire UTC day):
	// Both P1 and P2 fall inside this UTC day. P3 is outside (Aug 16).
	// Under Asia/Kolkata session timezone, P2's local representation would cross midnight into Aug 16,
	// BUT because our queries enforce UTC timestamp boundaries on TIMESTAMP WITHOUT TIME ZONE columns,
	// P1 and P2 MUST match identically under both UTC and Asia/Kolkata session timezones!
	fromUTC := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(2026, 8, 15, 23, 59, 59, 999999999, time.UTC)

	filter := domain.AdminPaymentFilter{
		From: &fromUTC,
		To:   &toUTC,
	}

	// 1. Execute under session timezone 'UTC'
	require.NoError(t, pinnedDB.Exec("SET TIME ZONE 'UTC'").Error)
	resPaymentsUTC, err := repo.FindAdminPayments(ctx, filter)
	require.NoError(t, err)
	resSummariesUTC, err := repo.GetEventPaymentSummaries(ctx, filter)
	require.NoError(t, err)

	assert.Equal(t, int64(2), resPaymentsUTC.TotalRecords, "UTC session must find exactly 2 payments (P1 and P2)")
	assert.Equal(t, int64(1), resSummariesUTC.TotalRecords, "UTC session must find exactly 1 event")
	assert.True(t, decimal.NewFromInt(1000).Equal(resSummariesUTC.OverallSummary.GrossCapturedAmount))

	// 2. Execute under session timezone 'Asia/Kolkata'
	require.NoError(t, pinnedDB.Exec("SET TIME ZONE 'Asia/Kolkata'").Error)
	resPaymentsKolkata, err := repo.FindAdminPayments(ctx, filter)
	require.NoError(t, err)
	resSummariesKolkata, err := repo.GetEventPaymentSummaries(ctx, filter)
	require.NoError(t, err)

	// 3. Verify identical results between UTC and Asia/Kolkata sessions
	assert.Equal(t, resPaymentsUTC.TotalRecords, resPaymentsKolkata.TotalRecords)
	assert.Equal(t, len(resPaymentsUTC.Payments), len(resPaymentsKolkata.Payments))
	for i := range resPaymentsUTC.Payments {
		assert.Equal(t, resPaymentsUTC.Payments[i].ID, resPaymentsKolkata.Payments[i].ID)
		assert.Equal(t, resPaymentsUTC.Payments[i].Amount, resPaymentsKolkata.Payments[i].Amount)
		assert.Equal(t, resPaymentsUTC.Payments[i].Status, resPaymentsKolkata.Payments[i].Status)
	}

	assert.Equal(t, resSummariesUTC.TotalRecords, resSummariesKolkata.TotalRecords)
	assert.Equal(t, resSummariesUTC.OverallSummary.TotalEvents, resSummariesKolkata.OverallSummary.TotalEvents)
	assert.True(t, resSummariesUTC.OverallSummary.GrossCapturedAmount.Equal(resSummariesKolkata.OverallSummary.GrossCapturedAmount))
	assert.Equal(t, resSummariesUTC.OverallSummary.RefundRequiredCount, resSummariesKolkata.OverallSummary.RefundRequiredCount)
	assert.Equal(t, resSummariesUTC.OverallSummary.UnallocatedCapturedCount, resSummariesKolkata.OverallSummary.UnallocatedCapturedCount)

	require.Len(t, resSummariesKolkata.Summaries, 1)
	assert.Equal(t, resSummariesUTC.Summaries[0].EventID, resSummariesKolkata.Summaries[0].EventID)
	assert.Equal(t, resSummariesUTC.Summaries[0].TotalPaymentsCount, resSummariesKolkata.Summaries[0].TotalPaymentsCount)
	assert.Equal(t, resSummariesUTC.Summaries[0].SuccessfullyCompletedPaymentsCount, resSummariesKolkata.Summaries[0].SuccessfullyCompletedPaymentsCount)
	assert.True(t, resSummariesUTC.Summaries[0].GrossCapturedAmount.Equal(resSummariesKolkata.Summaries[0].GrossCapturedAmount))

	// Reset timezone
	_ = pinnedDB.Exec("SET TIME ZONE 'UTC'")
}

func TestPaymentRepository_AllocationStatus_TaxAndDiscount_Regression(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, full_name, email) VALUES (?, 'Test Customer', 'customer@example.com')
	`, userID).Error)

	orgUserID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, full_name, email) VALUES (?, 'Test Org User', 'orguser@example.com')
	`, orgUserID).Error)

	orgID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO organizers (id, user_id) VALUES (?, ?)
	`, orgID, orgUserID).Error)

	orgProfID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Tax Regression Events LLC')
	`, orgProfID, orgID).Error)

	eventID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Tax & Discount Fest 2026')
	`, eventID, orgID, uuid.New()).Error)

	now := time.Now().UTC().Truncate(time.Second)
	acc1010, acc2010, acc2030, acc4010, _ := setupAdminFinanceAccountsWithTax(t, db)

	insertJournal := func(jID uuid.UUID, payID uuid.UUID, key string, isReversed bool, entries []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}) {
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`
				INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, created_at)
				VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, ?, ?, 'INR', 'Capture Journal', 'DRAFT', FALSE, ?)
			`, jID, payID.String(), key, "fp_"+key, now).Error; err != nil {
				return err
			}
			for _, e := range entries {
				if err := tx.Exec(`
					INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
					VALUES (?, ?, ?, ?, ?, 'INR', ?)
				`, uuid.New(), jID, e.accID, e.eType, e.amount, now).Error; err != nil {
					return err
				}
			}
			if err := tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, jID).Error; err != nil {
				return err
			}
			if isReversed {
				revJID := uuid.New()
				if err := tx.Exec(`
					INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, is_reversed, reversal_reference_id, created_at)
					VALUES (?, 'REVERSAL', 'JOURNAL_REVERSED', 'PAYMENT', ?, ?, ?, 'INR', 'Reversal Journal', 'DRAFT', FALSE, ?, ?)
				`, revJID, payID.String(), "rev_"+key, "fp_rev_"+key, jID, now).Error; err != nil {
					return err
				}
				for _, e := range entries {
					revType := "DEBIT"
					if e.eType == "DEBIT" {
						revType = "CREDIT"
					}
					if err := tx.Exec(`
						INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
						VALUES (?, ?, ?, ?, ?, 'INR', ?)
					`, uuid.New(), revJID, e.accID, revType, e.amount, now).Error; err != nil {
						return err
					}
				}
				if err := tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = ? WHERE id = ?`, now, revJID).Error; err != nil {
					return err
				}
				return tx.Exec(`UPDATE finance_journals SET is_reversed = TRUE, reversed_by_journal_id = ?, reversed_at = ? WHERE id = ?`, revJID, now, jID).Error
			}
			return nil
		})
		require.NoError(t, err)
	}

	// 1. Valid posted journal with tax -> ACCOUNTED
	// Booking: Subtotal 1000, Platform Fee 50, Tax 180, Total 1230
	bkTaxValid := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-01', ?, ?, 1000.00, 50.00, 180.00, 0.00, 1230.00, 'CONFIRMED')
	`, bkTaxValid, userID, eventID).Error)

	payTaxValid := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_01', 'ord_reg_01', 'CARD', 1230.00, 'SUCCESS', ?, ?, ?)
	`, payTaxValid, bkTaxValid, now, now, now).Error)

	insertJournal(uuid.New(), payTaxValid, "key_reg_01", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1230.00},
		{acc2010, "CREDIT", 1000.00},
		{acc4010, "CREDIT", 50.00},
		{acc2030, "CREDIT", 180.00},
	})

	// 2. Missing tax journal leg -> UNALLOCATED
	// Booking has Tax 180, but journal credits 4010 with 230 and omits 2030
	bkTaxMissing := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-02', ?, ?, 1000.00, 50.00, 180.00, 0.00, 1230.00, 'CONFIRMED')
	`, bkTaxMissing, userID, eventID).Error)

	payTaxMissing := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_02', 'ord_reg_02', 'CARD', 1230.00, 'SUCCESS', ?, ?, ?)
	`, payTaxMissing, bkTaxMissing, now, now, now).Error)

	insertJournal(uuid.New(), payTaxMissing, "key_reg_02", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1230.00},
		{acc2010, "CREDIT", 1000.00},
		{acc4010, "CREDIT", 230.00},
	})

	// 3. Incorrect tax amount -> UNALLOCATED
	// Booking has Tax 180, but journal credits 2030 with 150 and 4010 with 80
	bkTaxWrong := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-03', ?, ?, 1000.00, 50.00, 180.00, 0.00, 1230.00, 'CONFIRMED')
	`, bkTaxWrong, userID, eventID).Error)

	payTaxWrong := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_03', 'ord_reg_03', 'CARD', 1230.00, 'SUCCESS', ?, ?, ?)
	`, payTaxWrong, bkTaxWrong, now, now, now).Error)

	insertJournal(uuid.New(), payTaxWrong, "key_reg_03", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1230.00},
		{acc2010, "CREDIT", 1000.00},
		{acc4010, "CREDIT", 80.00},
		{acc2030, "CREDIT", 150.00},
	})

	// 4. Unbalanced journal -> UNALLOCATED
	bkUnbalanced := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-04', ?, ?, 1000.00, 50.00, 180.00, 0.00, 1230.00, 'CONFIRMED')
	`, bkUnbalanced, userID, eventID).Error)

	payUnbalanced := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_04', 'ord_reg_04', 'CARD', 1230.00, 'SUCCESS', ?, ?, ?)
	`, payUnbalanced, bkUnbalanced, now, now, now).Error)

	require.NoError(t, db.Exec(`ALTER TABLE finance_journals DISABLE TRIGGER trg_validate_journal_double_entry`).Error)
	insertJournal(uuid.New(), payUnbalanced, "key_reg_04", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1230.00},
		{acc2010, "CREDIT", 1000.00},
		{acc4010, "CREDIT", 50.00},
		{acc2030, "CREDIT", 100.00},
	})
	require.NoError(t, db.Exec(`ALTER TABLE finance_journals ENABLE TRIGGER trg_validate_journal_double_entry`).Error)

	// 5. Discounted booking with correct net organizer payable -> ACCOUNTED
	// Subtotal 1000, Discount 200, Fee 50, Tax 180, Total 1030
	// Net Organizer Revenue = Subtotal - Discount = 800.00
	bkDiscounted := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-05', ?, ?, 1000.00, 50.00, 180.00, 200.00, 1030.00, 'CONFIRMED')
	`, bkDiscounted, userID, eventID).Error)

	payDiscounted := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_05', 'ord_reg_05', 'CARD', 1030.00, 'SUCCESS', ?, ?, ?)
	`, payDiscounted, bkDiscounted, now, now, now).Error)

	insertJournal(uuid.New(), payDiscounted, "key_reg_05", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1030.00},
		{acc2010, "CREDIT", 800.00},
		{acc4010, "CREDIT", 50.00},
		{acc2030, "CREDIT", 180.00},
	})

	// 6. Reversed journal -> UNALLOCATED
	bkReversed := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-06', ?, ?, 1000.00, 50.00, 180.00, 0.00, 1230.00, 'CONFIRMED')
	`, bkReversed, userID, eventID).Error)

	payReversed := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_06', 'ord_reg_06', 'CARD', 1230.00, 'SUCCESS', ?, ?, ?)
	`, payReversed, bkReversed, now, now, now).Error)

	insertJournal(uuid.New(), payReversed, "key_reg_06", true, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 1230.00},
		{acc2010, "CREDIT", 1000.00},
		{acc4010, "CREDIT", 50.00},
		{acc2030, "CREDIT", 180.00},
	})

	// 7. Zero-tax booking -> ACCOUNTED
	// Subtotal 500, Platform Fee 25, Tax 0, Total 525
	bkZeroTax := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status)
		VALUES (?, 'BK-REG-07', ?, ?, 500.00, 25.00, 0.00, 0.00, 525.00, 'CONFIRMED')
	`, bkZeroTax, userID, eventID).Error)

	payZeroTax := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?, ?, 'pay_reg_07', 'ord_reg_07', 'UPI', 525.00, 'SUCCESS', ?, ?, ?)
	`, payZeroTax, bkZeroTax, now, now, now).Error)

	insertJournal(uuid.New(), payZeroTax, "key_reg_07", false, []struct {
		accID  uuid.UUID
		eType  string
		amount float64
	}{
		{acc1010, "DEBIT", 525.00},
		{acc2010, "CREDIT", 500.00},
		{acc4010, "CREDIT", 25.00},
	})

	// Execute FindAdminPayments
	res, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
		Limit: 50,
	})
	require.NoError(t, err)

	payMap := make(map[string]domain.AdminPaymentListItem)
	for _, item := range res.Payments {
		payMap[item.ID] = item
	}

	// 1. Valid posted journal with tax -> ACCOUNTED
	p1, ok1 := payMap[payTaxValid.String()]
	require.True(t, ok1)
	assert.Equal(t, domain.AllocationStatusAccounted, p1.AllocationStatus, "Valid posted journal with tax must be ACCOUNTED")

	// 2. Missing tax journal leg -> UNALLOCATED
	p2, ok2 := payMap[payTaxMissing.String()]
	require.True(t, ok2)
	assert.Equal(t, domain.AllocationStatusUnallocated, p2.AllocationStatus, "Missing tax journal leg must be UNALLOCATED")

	// 3. Incorrect tax amount -> UNALLOCATED
	p3, ok3 := payMap[payTaxWrong.String()]
	require.True(t, ok3)
	assert.Equal(t, domain.AllocationStatusUnallocated, p3.AllocationStatus, "Incorrect tax amount must be UNALLOCATED")

	// 4. Unbalanced journal -> UNALLOCATED
	p4, ok4 := payMap[payUnbalanced.String()]
	require.True(t, ok4)
	assert.Equal(t, domain.AllocationStatusUnallocated, p4.AllocationStatus, "Unbalanced journal must be UNALLOCATED")

	// 5. Discounted booking with correct net organizer payable -> ACCOUNTED
	p5, ok5 := payMap[payDiscounted.String()]
	require.True(t, ok5)
	assert.Equal(t, domain.AllocationStatusAccounted, p5.AllocationStatus, "Discounted booking with correct net organizer payable must be ACCOUNTED")

	// 6. Reversed journal -> UNALLOCATED
	p6, ok6 := payMap[payReversed.String()]
	require.True(t, ok6)
	assert.Equal(t, domain.AllocationStatusUnallocated, p6.AllocationStatus, "Reversed journal must be UNALLOCATED")

	// 7. Zero-tax booking -> ACCOUNTED
	p7, ok7 := payMap[payZeroTax.String()]
	require.True(t, ok7)
	assert.Equal(t, domain.AllocationStatusAccounted, p7.AllocationStatus, "Zero-tax booking with valid journal must be ACCOUNTED")

	// Also verify Organizer repository produces identical results for this organizer
	orgRepo := repository.NewPaymentRepository(db, slog.Default())
	orgRes, err := orgRepo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
		OrganizerID: orgID,
		Limit:       50,
	})
	require.NoError(t, err)
	orgPayMap := make(map[string]domain.OrganizerPaymentListItem)
	for _, item := range orgRes.Payments {
		orgPayMap[item.ID] = item
	}

	assert.Equal(t, domain.AllocationStatusAccounted, orgPayMap[payTaxValid.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusUnallocated, orgPayMap[payTaxMissing.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusUnallocated, orgPayMap[payTaxWrong.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusUnallocated, orgPayMap[payUnbalanced.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusAccounted, orgPayMap[payDiscounted.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusUnallocated, orgPayMap[payReversed.String()].AllocationStatus)
	assert.Equal(t, domain.AllocationStatusAccounted, orgPayMap[payZeroTax.String()].AllocationStatus)
}

func TestPaymentRepository_RefundVisibility_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	now := time.Now().UTC()
	userID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Refund Customer', 'refund_cust@example.com')`, userID).Error)

	orgUserID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Refund Org User', 'refund_org@example.com')`, orgUserID).Error)
	orgID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID, orgUserID).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Refund Visibility Productions')`, uuid.New(), orgID).Error)

	eventID := uuid.New()
	categoryID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO events (id, organizer_id, category_id, title, status)
		VALUES (?, ?, ?, 'Refund Test Event', 'PUBLISHED')
	`, eventID, orgID, categoryID).Error)

	// Helper to insert booking & payment
	createBookingAndPayment := func(bookingNum string, amount float64) (uuid.UUID, uuid.UUID) {
		bID := uuid.New()
		pID := uuid.New()
		txID := "tx_" + pID.String()[:8]
		orderID := "order_" + pID.String()[:8]

		require.NoError(t, db.Exec(`
			INSERT INTO bookings (id, user_id, event_id, booking_number, status, subtotal, platform_fee, tax, total, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'CONFIRMED', ?, 0.00, 0.00, ?, ?, ?)
		`, bID, userID, eventID, bookingNum, amount, amount, now, now).Error)

		require.NoError(t, db.Exec(`
			INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'UPI', ?, 'SUCCESS', ?, ?, ?)
		`, pID, bID, txID, orderID, amount, now, now, now).Error)

		return bID, pID
	}

	// 1. Payment with NO refunds (Amount: ₹125.00)
	_, payNoRefund := createBookingAndPayment("BK-NO-REFUND", 125.00)

	// 2. Partial successful refund (Amount: ₹125.00, Refund: ₹50.00 SUCCESS)
	bPart, payPartial := createBookingAndPayment("BK-PARTIAL", 125.00)
	rfndPartID := uuid.New()
	gwPartID := "rfnd_part_50"
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 50.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', ?, ?, ?)
	`, rfndPartID, bPart, payPartial, gwPartID, now, now, now).Error)

	// 3. Full successful refund (Amount: ₹200.00, Refund: ₹200.00 SUCCESS)
	bFull, payFull := createBookingAndPayment("BK-FULL", 200.00)
	rfndFullID := uuid.New()
	gwFullID := "rfnd_full_200"
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 200.00, 'SUCCESS', 'EVENT_CANCELLATION', ?, ?, ?)
	`, rfndFullID, bFull, payFull, gwFullID, now, now, now).Error)

	// 4. Multiple successful partial refunds (Amount: ₹1000.00, 2x ₹300.00 SUCCESS)
	bMulti, payMulti := createBookingAndPayment("BK-MULTI", 1000.00)
	gwMulti1 := "rfnd_multi_1"
	gwMulti2 := "rfnd_multi_2"
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 300.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', ?, ?, ?)
	`, uuid.New(), bMulti, payMulti, gwMulti1, now, now, now).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 300.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', ?, ?, ?)
	`, uuid.New(), bMulti, payMulti, gwMulti2, now, now, now).Error)

	// 5. Pending refund (Amount: ₹300.00, 1x ₹100.00 PENDING)
	bPending, payPending := createBookingAndPayment("BK-PENDING", 300.00)
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, created_at, updated_at)
		VALUES (?, ?, ?, NULL, 100.00, 'PENDING', 'CUSTOMER_CANCELLATION', ?, ?)
	`, uuid.New(), bPending, payPending, now, now).Error)

	// 6. Failed refund (Amount: ₹400.00, 1x ₹100.00 FAILED)
	bFailed, payFailed := createBookingAndPayment("BK-FAILED", 400.00)
	gwFailed := "rfnd_failed_100"
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, 100.00, 'FAILED', 'GATEWAY_ERROR', ?, ?)
	`, uuid.New(), bFailed, payFailed, gwFailed, now, now).Error)

	// 7. Partial successful refund with another pending refund (Amount: ₹500.00, ₹150.00 SUCCESS + ₹100.00 PENDING)
	bPartPend, payPartPend := createBookingAndPayment("BK-PART-PEND", 500.00)
	gwPartPendSucc := "rfnd_part_succ_150"
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 150.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', ?, ?, ?)
	`, uuid.New(), bPartPend, payPartPend, gwPartPendSucc, now, now, now).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, created_at, updated_at)
		VALUES (?, ?, ?, NULL, 100.00, 'PENDING', 'CUSTOMER_CANCELLATION', ?, ?)
	`, uuid.New(), bPartPend, payPartPend, now, now).Error)

	// --- Execute Admin Query ---
	adminRes, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
		EventID: &eventID,
		Limit:   50,
	})
	require.NoError(t, err)
	require.Len(t, adminRes.Payments, 7)

	adminMap := make(map[string]domain.AdminPaymentListItem)
	for _, p := range adminRes.Payments {
		adminMap[p.ID] = p
	}

	// --- Verify Admin Results ---
	t.Run("1. Payment with no refunds", func(t *testing.T) {
		p := adminMap[payNoRefund.String()]
		assert.True(t, decimal.NewFromInt(125).Equal(p.Amount))
		assert.True(t, decimal.Zero.Equal(p.RefundedAmount))
		assert.True(t, decimal.Zero.Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(125).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusNone, p.RefundStatus)
		assert.Equal(t, 0, p.RefundCount)
		assert.Empty(t, p.Refunds)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status, "Original payment status must remain SUCCESS")
	})

	t.Run("2. Partial successful refund", func(t *testing.T) {
		p := adminMap[payPartial.String()]
		assert.True(t, decimal.NewFromInt(125).Equal(p.Amount))
		assert.True(t, decimal.NewFromInt(50).Equal(p.RefundedAmount))
		assert.True(t, decimal.Zero.Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(75).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusPartial, p.RefundStatus)
		assert.Equal(t, 1, p.RefundCount)
		require.Len(t, p.Refunds, 1)
		assert.Equal(t, rfndPartID, p.Refunds[0].ID)
		assert.Equal(t, &gwPartID, p.Refunds[0].GatewayRefundID)
		assert.True(t, decimal.NewFromInt(50).Equal(p.Refunds[0].Amount))
		assert.Equal(t, "SUCCESS", p.Refunds[0].Status)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	t.Run("3. Full successful refund", func(t *testing.T) {
		p := adminMap[payFull.String()]
		assert.True(t, decimal.NewFromInt(200).Equal(p.Amount))
		assert.True(t, decimal.NewFromInt(200).Equal(p.RefundedAmount))
		assert.True(t, decimal.Zero.Equal(p.PendingRefundAmount))
		assert.True(t, decimal.Zero.Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusFull, p.RefundStatus)
		assert.Equal(t, 1, p.RefundCount)
		require.Len(t, p.Refunds, 1)
		assert.Equal(t, "SUCCESS", p.Refunds[0].Status)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	t.Run("4. Multiple successful partial refunds", func(t *testing.T) {
		p := adminMap[payMulti.String()]
		assert.True(t, decimal.NewFromInt(1000).Equal(p.Amount))
		assert.True(t, decimal.NewFromInt(600).Equal(p.RefundedAmount))
		assert.True(t, decimal.Zero.Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(400).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusPartial, p.RefundStatus)
		assert.Equal(t, 2, p.RefundCount)
		require.Len(t, p.Refunds, 2)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	t.Run("5. Pending refund does not reduce net amount", func(t *testing.T) {
		p := adminMap[payPending.String()]
		assert.True(t, decimal.NewFromInt(300).Equal(p.Amount))
		assert.True(t, decimal.Zero.Equal(p.RefundedAmount))
		assert.True(t, decimal.NewFromInt(100).Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(300).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusPending, p.RefundStatus)
		assert.Equal(t, 1, p.RefundCount)
		require.Len(t, p.Refunds, 1)
		assert.Equal(t, "PENDING", p.Refunds[0].Status)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	t.Run("6. Failed refund does not reduce net amount", func(t *testing.T) {
		p := adminMap[payFailed.String()]
		assert.True(t, decimal.NewFromInt(400).Equal(p.Amount))
		assert.True(t, decimal.Zero.Equal(p.RefundedAmount))
		assert.True(t, decimal.Zero.Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(400).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusFailed, p.RefundStatus)
		assert.Equal(t, 1, p.RefundCount)
		require.Len(t, p.Refunds, 1)
		assert.Equal(t, "FAILED", p.Refunds[0].Status)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	t.Run("7. Partial successful refund with another pending refund", func(t *testing.T) {
		p := adminMap[payPartPend.String()]
		assert.True(t, decimal.NewFromInt(500).Equal(p.Amount))
		assert.True(t, decimal.NewFromInt(150).Equal(p.RefundedAmount))
		assert.True(t, decimal.NewFromInt(100).Equal(p.PendingRefundAmount))
		assert.True(t, decimal.NewFromInt(350).Equal(p.NetAmount))
		assert.Equal(t, domain.PaymentRefundStatusPartial, p.RefundStatus)
		assert.Equal(t, 2, p.RefundCount)
		require.Len(t, p.Refunds, 2)
		assert.Equal(t, domain.PaymentStatusSuccess, p.Status)
	})

	// --- 8. Admin/Organizer API response parity ---
	t.Run("8. Admin and Organizer API response parity", func(t *testing.T) {
		orgRes, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgID,
			EventID:     &eventID,
			Limit:       50,
		})
		require.NoError(t, err)
		require.Len(t, orgRes.Payments, 7)

		orgMap := make(map[string]domain.OrganizerPaymentListItem)
		for _, p := range orgRes.Payments {
			orgMap[p.ID] = p
		}

		for id, ap := range adminMap {
			op, exists := orgMap[id]
			require.True(t, exists, "Payment %s must exist in Organizer results", id)
			assert.True(t, ap.Amount.Equal(op.Amount))
			assert.True(t, ap.RefundedAmount.Equal(op.RefundedAmount))
			assert.True(t, ap.PendingRefundAmount.Equal(op.PendingRefundAmount))
			assert.True(t, ap.NetAmount.Equal(op.NetAmount))
			assert.Equal(t, ap.RefundStatus, op.RefundStatus)
			assert.Equal(t, ap.RefundCount, op.RefundCount)
			assert.Equal(t, len(ap.Refunds), len(op.Refunds))
		}
	})

	// --- 9. Pagination and filtering remain correct ---
	t.Run("9. Pagination and filtering remain correct", func(t *testing.T) {
		page1, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			EventID: &eventID,
			Page:    1,
			Limit:   4,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(7), page1.TotalRecords)
		assert.Len(t, page1.Payments, 4)

		page2, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			EventID: &eventID,
			Page:    2,
			Limit:   4,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(7), page2.TotalRecords)
		assert.Len(t, page2.Payments, 3)

		// Search filtering
		searchRes, err := repo.FindAdminPayments(ctx, domain.AdminPaymentFilter{
			EventID: &eventID,
			Search:  "BK-PARTIAL",
		})
		require.NoError(t, err)
		require.Len(t, searchRes.Payments, 1)
		assert.Equal(t, "BK-PARTIAL", searchRes.Payments[0].BookingNumber)
		assert.True(t, decimal.NewFromInt(50).Equal(searchRes.Payments[0].RefundedAmount))
	})
}

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

func setupAdminDashboardTestTables(t *testing.T, db *gorm.DB) {
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

	CREATE TABLE IF NOT EXISTS settlements (
		id UUID PRIMARY KEY,
		organizer_id UUID NOT NULL,
		event_id UUID,
		amount NUMERIC(12, 2) NOT NULL,
		currency VARCHAR(3) NOT NULL DEFAULT 'INR',
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		payout_reference VARCHAR(255),
		failure_reason TEXT,
		rejection_reason TEXT,
		approved_by UUID,
		approved_at TIMESTAMPTZ,
		rejected_at TIMESTAMPTZ,
		processed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_settlements_organizer_id ON settlements(organizer_id);
	`

	require.NoError(t, db.Exec(ddl).Error)

	truncateSQL := `
		TRUNCATE TABLE 
			settlements,
			refunds, 
			payments, 
			booking_items,
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

func seedAdminFinanceAccounts(t *testing.T, db *gorm.DB) (acc1010, acc2010, acc2030, acc4010, acc1020 uuid.UUID) {
	t.Helper()
	acc1010 = uuid.New()
	acc2010 = uuid.New()
	acc2030 = uuid.New()
	acc4010 = uuid.New()
	acc1020 = uuid.New()

	err := db.Exec(`
		INSERT INTO finance_accounts (id, code, name, type, normal_balance, currency, is_active, created_at)
		VALUES 
			(?, '1010', 'Razorpay Clearing', 'ASSET', 'DEBIT', 'INR', TRUE, NOW()),
			(?, '2010', 'Organizer Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '2030', 'Tax Payable', 'LIABILITY', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '4010', 'Platform Fee Revenue', 'REVENUE', 'CREDIT', 'INR', TRUE, NOW()),
			(?, '1020', 'Operating Bank', 'ASSET', 'DEBIT', 'INR', TRUE, NOW())
	`, acc1010, acc2010, acc2030, acc4010, acc1020).Error
	require.NoError(t, err)

	return acc1010, acc2010, acc2030, acc4010, acc1020
}

func TestAdminDashboardRepository_EmptyDatabase(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminDashboardTestTables(t, db)
	repo := repository.NewAdminDashboardRepository(db, slog.Default())
	ctx := context.Background()

	// 1. Platform overview on empty DB
	po, err := repo.GetPlatformOverview(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), po.TotalUsers)
	assert.Equal(t, int64(0), po.TotalOrganizers)
	assert.Equal(t, int64(0), po.TotalEvents)
	assert.Equal(t, int64(0), po.ActiveEvents)
	assert.Equal(t, int64(0), po.TotalBookings)
	assert.Equal(t, int64(0), po.TicketsSold)

	// 2. Financial overview on empty DB
	fo, err := repo.GetFinancialOverview(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)
	assert.True(t, fo.GrossCapturedAmount.IsZero())
	assert.True(t, fo.PlatformFeeRevenue.IsZero())
	assert.True(t, fo.OrganizerPayableLiability.IsZero())
	assert.True(t, fo.TotalRefunds.IsZero())
	assert.True(t, fo.CompletedSettlements.IsZero())
	assert.True(t, fo.PendingSettlements.IsZero())
	assert.True(t, fo.TaxLiability.IsZero())
	assert.Equal(t, "INR", fo.Currency)

	// 3. Trends on empty DB
	trends, err := repo.GetTrends(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)
	assert.NotEmpty(t, trends.RevenueTrends)
	assert.NotEmpty(t, trends.BookingTrends)
	assert.NotEmpty(t, trends.RefundTrends)
	for _, rt := range trends.RevenueTrends {
		assert.True(t, rt.GrossVolume.IsZero())
		assert.True(t, rt.PlatformFees.IsZero())
	}

	// 4. Payment status breakdown on empty DB
	breakdown, err := repo.GetPaymentStatusBreakdown(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)
	assert.Len(t, breakdown, 4) // SUCCESS, PENDING, FAILED, REFUND_REQUIRED
	for _, item := range breakdown {
		assert.Equal(t, int64(0), item.Count)
		assert.True(t, item.Amount.IsZero())
	}

	// 5. Recent bookings on empty DB
	recent, err := repo.GetRecentBookings(ctx, 5)
	require.NoError(t, err)
	assert.Empty(t, recent)
}

func TestAdminDashboardRepository_PopulatedData_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminDashboardTestTables(t, db)
	repo := repository.NewAdminDashboardRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, acc2010, acc2030, acc4010, acc1020 := seedAdminFinanceAccounts(t, db)

	// Seed Users: 1 Organizer, 2 Customers
	userOrg := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email, role) VALUES (?, 'Org Owner', 'org@test.com', 'ORGANIZER')`, userOrg).Error)

	cust1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email, role) VALUES (?, 'Customer One', 'c1@test.com', 'CUSTOMER')`, cust1).Error)

	cust2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email, role) VALUES (?, 'Customer Two', 'c2@test.com', 'CUSTOMER')`, cust2).Error)

	// Seed Organizer & Profile
	orgID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id, status) VALUES (?, ?, 'APPROVED')`, orgID, userOrg).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Starlight Events')`, uuid.New(), orgID).Error)

	// Seed Events: 1 PUBLISHED, 1 DRAFT
	event1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, status) VALUES (?, ?, ?, 'Rock Concert', 'PUBLISHED')`, event1, orgID, uuid.New()).Error)

	event2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title, status) VALUES (?, ?, ?, 'Private Meetup', 'DRAFT')`, event2, orgID, uuid.New()).Error)

	// Seed Bookings
	// Booking 1: Confirmed with 3 tickets (subtotal 1000, fee 100, tax 80, total 1180)
	booking1 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, total, status, created_at)
		VALUES (?, 'BK-001', ?, ?, 1000, 100, 80, 1180, 'CONFIRMED', NOW() - INTERVAL '2 days')
	`, booking1, cust1, event1).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price, created_at, updated_at)
		VALUES (?, ?, 3, 333.33, 1000, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days')
	`, uuid.New(), booking1).Error)

	// Booking 2: Pending with 2 tickets (subtotal 500, fee 50, tax 0, total 550)
	booking2 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, total, status, created_at)
		VALUES (?, 'BK-002', ?, ?, 500, 50, 0, 550, 'PENDING', NOW() - INTERVAL '1 day')
	`, booking2, cust2, event1).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price, created_at, updated_at)
		VALUES (?, ?, 2, 250, 500, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day')
	`, uuid.New(), booking2).Error)

	// Seed Payments:
	// Pay 1: SUCCESS for Booking 1 (1180)
	pay1ID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, amount, status, paid_at, created_at)
		VALUES (?, ?, 'txn_1001', 'order_1001', 1180, 'SUCCESS', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days')
	`, pay1ID, booking1).Error)

	// Pay 2: PENDING for Booking 2 (550)
	pay2ID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider_order_id, amount, status, paid_at, created_at)
		VALUES (?, ?, 'txn_1002', 'order_1002', 550, 'PENDING', NULL, NOW() - INTERVAL '1 day')
	`, pay2ID, booking2).Error)

	// Seed Posted Financial Journal for Pay 1:
	// Debit 1010 (1180), Credit 2010 (1000), Credit 4010 (100), Credit 2030 (80)
	journal1 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_p1', 'fp_p1', 'INR', 'Sale capture', 'DRAFT', NOW() - INTERVAL '2 days', FALSE)
	`, journal1, pay1ID.String()).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 1180, 'INR', ?, ?, NOW() - INTERVAL '2 days'),
			(?, ?, ?, 'CREDIT', 1000, 'INR', ?, ?, NOW() - INTERVAL '2 days'),
			(?, ?, ?, 'CREDIT', 100, 'INR', ?, ?, NOW() - INTERVAL '2 days'),
			(?, ?, ?, 'CREDIT', 80, 'INR', ?, ?, NOW() - INTERVAL '2 days')
	`,
		uuid.New(), journal1, acc1010, orgID, event1,
		uuid.New(), journal1, acc2010, orgID, event1,
		uuid.New(), journal1, acc4010, orgID, event1,
		uuid.New(), journal1, acc2030, orgID, event1,
	).Error)
	require.NoError(t, db.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() - INTERVAL '2 days' WHERE id = ?`, journal1).Error)

	// Seed Refund: 1 SUCCESS refund of 236 on Pay 1
	refund1ID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, gateway_refund_id, amount, status, reason, created_at, processed_at)
		VALUES (?, ?, ?, 'rfnd_001', 236, 'SUCCESS', 'CUSTOMER_CANCELLATION', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day')
	`, refund1ID, booking1, pay1ID).Error)

	// Posted Refund Journal:
	// Debit 2010 (200), Debit 4010 (20), Debit 2030 (16), Credit 1010 (236)
	refJournal := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'REFUND', 'CUSTOMER_REFUND', 'REFUND', ?, 'key_r1', 'fp_r1', 'INR', 'Refund processed', 'DRAFT', NOW() - INTERVAL '1 day', FALSE)
	`, refJournal, refund1ID.String()).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 200, 'INR', ?, ?, NOW() - INTERVAL '1 day'),
			(?, ?, ?, 'DEBIT', 20, 'INR', ?, ?, NOW() - INTERVAL '1 day'),
			(?, ?, ?, 'DEBIT', 16, 'INR', ?, ?, NOW() - INTERVAL '1 day'),
			(?, ?, ?, 'CREDIT', 236, 'INR', ?, ?, NOW() - INTERVAL '1 day')
	`,
		uuid.New(), refJournal, acc2010, orgID, event1,
		uuid.New(), refJournal, acc4010, orgID, event1,
		uuid.New(), refJournal, acc2030, orgID, event1,
		uuid.New(), refJournal, acc1010, orgID, event1,
	).Error)
	require.NoError(t, db.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() - INTERVAL '1 day' WHERE id = ?`, refJournal).Error)

	// Seed Settlements:
	// 1 COMPLETED of 500
	settlement1ID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO settlements (id, organizer_id, event_id, amount, status, processed_at, created_at)
		VALUES (?, ?, ?, 500, 'COMPLETED', NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours')
	`, settlement1ID, orgID, event1).Error)

	// Settlement Journal: Debit 2010 (500), Credit 1020 (500)
	settleJournal := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'SETTLEMENT', 'ORGANIZER_PAYOUT_SETTLED', 'SETTLEMENT', ?, 'key_s1', 'fp_s1', 'INR', 'Settlement completed', 'DRAFT', NOW() - INTERVAL '12 hours', FALSE)
	`, settleJournal, settlement1ID.String()).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 500, 'INR', ?, ?, NOW() - INTERVAL '12 hours'),
			(?, ?, ?, 'CREDIT', 500, 'INR', ?, ?, NOW() - INTERVAL '12 hours')
	`,
		uuid.New(), settleJournal, acc2010, orgID, event1,
		uuid.New(), settleJournal, acc1020, orgID, event1,
	).Error)
	require.NoError(t, db.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() - INTERVAL '12 hours' WHERE id = ?`, settleJournal).Error)

	// 1 In-Flight Settlement (PENDING) of 250
	settlement2ID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO settlements (id, organizer_id, event_id, amount, status, created_at)
		VALUES (?, ?, ?, 250, 'PENDING', NOW() - INTERVAL '1 hour')
	`, settlement2ID, orgID, event1).Error)

	// -------------------------------------------------------------------------
	// Assertions
	// -------------------------------------------------------------------------

	// 1. Platform Overview
	po, err := repo.GetPlatformOverview(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), po.TotalUsers)
	assert.Equal(t, int64(1), po.TotalOrganizers)
	assert.Equal(t, int64(2), po.TotalEvents)
	assert.Equal(t, int64(1), po.ActiveEvents) // only 'Rock Concert' is PUBLISHED
	assert.Equal(t, int64(2), po.TotalBookings)
	assert.Equal(t, int64(3), po.TicketsSold) // only from CONFIRMED booking

	// 2. Financial Overview
	fo, err := repo.GetFinancialOverview(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)

	// Gross captured = 1180
	assert.True(t, decimal.NewFromInt(1180).Equal(fo.GrossCapturedAmount), "Gross captured must be 1180, got %s", fo.GrossCapturedAmount)
	// Platform Fee Revenue = 100 (Credit) - 20 (Debit) = 80
	assert.True(t, decimal.NewFromInt(80).Equal(fo.PlatformFeeRevenue), "Platform fee revenue must be 80, got %s", fo.PlatformFeeRevenue)
	// Tax Liability = 80 (Credit) - 16 (Debit) = 64
	assert.True(t, decimal.NewFromInt(64).Equal(fo.TaxLiability), "Tax liability must be 64, got %s", fo.TaxLiability)
	// Organizer Payable = 1000 (Credit) - 200 (Refund Debit) - 500 (Payout Debit) = 300
	assert.True(t, decimal.NewFromInt(300).Equal(fo.OrganizerPayableLiability), "Organizer payable must be 300, got %s", fo.OrganizerPayableLiability)
	// Total Refunds = 236
	assert.True(t, decimal.NewFromInt(236).Equal(fo.TotalRefunds), "Total refunds must be 236, got %s", fo.TotalRefunds)
	// Completed Settlements = 500
	assert.True(t, decimal.NewFromInt(500).Equal(fo.CompletedSettlements), "Completed settlements must be 500, got %s", fo.CompletedSettlements)
	// Pending Settlements = 250
	assert.True(t, decimal.NewFromInt(250).Equal(fo.PendingSettlements), "Pending settlements must be 250, got %s", fo.PendingSettlements)
	assert.Equal(t, "INR", fo.Currency)

	// 3. Payment Status Breakdown
	breakdown, err := repo.GetPaymentStatusBreakdown(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)
	breakdownMap := make(map[string]domain.AdminPaymentStatusBreakdownItem)
	for _, b := range breakdown {
		breakdownMap[b.Status] = b
	}

	assert.Equal(t, int64(1), breakdownMap["SUCCESS"].Count)
	assert.True(t, decimal.NewFromInt(1180).Equal(breakdownMap["SUCCESS"].Amount))

	assert.Equal(t, int64(1), breakdownMap["PENDING"].Count)
	assert.True(t, decimal.NewFromInt(550).Equal(breakdownMap["PENDING"].Amount))

	assert.Equal(t, int64(0), breakdownMap["FAILED"].Count)
	assert.True(t, breakdownMap["FAILED"].Amount.IsZero())

	assert.Equal(t, int64(0), breakdownMap["REFUND_REQUIRED"].Count)
	assert.True(t, breakdownMap["REFUND_REQUIRED"].Amount.IsZero())

	// 4. Recent Bookings
	recent, err := repo.GetRecentBookings(ctx, 5)
	require.NoError(t, err)
	require.Len(t, recent, 2)

	// Latest booking first: BK-002 (1 day ago) before BK-001 (2 days ago)
	assert.Equal(t, "BK-002", recent[0].BookingNumber)
	assert.Equal(t, "Customer Two", recent[0].CustomerName)
	assert.Equal(t, "c2@test.com", recent[0].CustomerEmail)
	assert.Equal(t, "Rock Concert", recent[0].EventTitle)
	assert.Equal(t, "PENDING", recent[0].Status)
	assert.Equal(t, "PENDING", recent[0].PaymentStatus)

	assert.Equal(t, "BK-001", recent[1].BookingNumber)
	assert.Equal(t, "Customer One", recent[1].CustomerName)
	assert.Equal(t, "c1@test.com", recent[1].CustomerEmail)
	assert.Equal(t, "Rock Concert", recent[1].EventTitle)
	assert.Equal(t, "CONFIRMED", recent[1].Status)
	assert.Equal(t, "SUCCESS", recent[1].PaymentStatus)
}

func TestAdminDashboardRepository_CompensatingReversal_NetsToZero_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminDashboardTestTables(t, db)
	repo := repository.NewAdminDashboardRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, acc2010, acc2030, acc4010, _ := seedAdminFinanceAccounts(t, db)

	orgID := uuid.New()
	eventID := uuid.New()

	// 1. Seed original posted journal
	origJournalID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (
			id, transaction_type, financial_event_type, source_type, source_id,
			idempotency_key, idempotency_fingerprint, currency, description,
			status, created_at, is_reversed
		) VALUES (
			?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', 'pay_orig',
			'key_orig', 'fp_orig', 'INR', 'Original sale',
			'DRAFT', NOW() - INTERVAL '5 days', FALSE
		)
	`, origJournalID).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 1150, 'INR', ?, ?, NOW() - INTERVAL '5 days'),
			(?, ?, ?, 'CREDIT', 1000, 'INR', ?, ?, NOW() - INTERVAL '5 days'),
			(?, ?, ?, 'CREDIT', 100, 'INR', ?, ?, NOW() - INTERVAL '5 days'),
			(?, ?, ?, 'CREDIT', 50, 'INR', ?, ?, NOW() - INTERVAL '5 days')
	`,
		uuid.New(), origJournalID, acc1010, orgID, eventID,
		uuid.New(), origJournalID, acc2010, orgID, eventID,
		uuid.New(), origJournalID, acc4010, orgID, eventID,
		uuid.New(), origJournalID, acc2030, orgID, eventID,
	).Error)
	require.NoError(t, db.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() - INTERVAL '5 days' WHERE id = ?`, origJournalID).Error)

	// 2. Seed compensating reversal journal in a transaction (trigger is checked at commit)
	revJournalID := uuid.New()
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (
				id, transaction_type, financial_event_type, source_type, source_id,
				idempotency_key, idempotency_fingerprint, currency, description,
				status, created_at, is_reversed, reversal_reference_id
			) VALUES (
				?, 'REVERSAL', 'TICKET_SALE_CAPTURED', 'PAYMENT', 'pay_orig',
				'key_rev', 'fp_rev', 'INR', 'Compensating reversal for orig',
				'DRAFT', NOW() - INTERVAL '4 days', FALSE, ?
			)
		`, revJournalID, origJournalID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
			VALUES 
				(?, ?, ?, 'CREDIT', 1150, 'INR', ?, ?, NOW() - INTERVAL '4 days'),
				(?, ?, ?, 'DEBIT', 1000, 'INR', ?, ?, NOW() - INTERVAL '4 days'),
				(?, ?, ?, 'DEBIT', 100, 'INR', ?, ?, NOW() - INTERVAL '4 days'),
				(?, ?, ?, 'DEBIT', 50, 'INR', ?, ?, NOW() - INTERVAL '4 days')
		`,
			uuid.New(), revJournalID, acc1010, orgID, eventID,
			uuid.New(), revJournalID, acc2010, orgID, eventID,
			uuid.New(), revJournalID, acc4010, orgID, eventID,
			uuid.New(), revJournalID, acc2030, orgID, eventID,
		).Error; err != nil {
			return err
		}

		if err := tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() - INTERVAL '4 days' WHERE id = ?`, revJournalID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE finance_journals SET is_reversed = TRUE, reversed_at = NOW() - INTERVAL '4 days', reversed_by_journal_id = ? WHERE id = ?`, revJournalID, origJournalID).Error
	}))

	// 3. Query Financial Overview - must strictly net to zero!
	fo, err := repo.GetFinancialOverview(ctx, domain.AdminDashboardFilter{})
	require.NoError(t, err)

	assert.True(t, fo.PlatformFeeRevenue.IsZero(), "Platform fee revenue must net to zero after reversal, got %s", fo.PlatformFeeRevenue)
	assert.True(t, fo.OrganizerPayableLiability.IsZero(), "Organizer payable liability must net to zero after reversal, got %s", fo.OrganizerPayableLiability)
	assert.True(t, fo.TaxLiability.IsZero(), "Tax liability must net to zero after reversal, got %s", fo.TaxLiability)
}

func TestAdminDashboardRepository_DateFilteringAndTrends_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupAdminDashboardTestTables(t, db)
	repo := repository.NewAdminDashboardRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, _, _, acc4010, _ := seedAdminFinanceAccounts(t, db)

	// Seed booking & payment inside target window (e.g. 2026-05)
	bookingID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, total, status, created_at)
		VALUES (?, 'BK-TREND-01', ?, ?, 1000, 'CONFIRMED', '2026-05-15 10:00:00+00')
	`, bookingID, uuid.New(), uuid.New()).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price, created_at, updated_at)
		VALUES (?, ?, 2, 500, 1000, '2026-05-15 10:00:00+00', '2026-05-15 10:00:00+00')
	`, uuid.New(), bookingID).Error)

	payID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, amount, status, paid_at, created_at)
		VALUES (?, ?, 1000, 'SUCCESS', '2026-05-15 10:05:00+00', '2026-05-15 10:00:00+00')
	`, payID, bookingID).Error)

	// Seed journal in 2026-05
	journalID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO finance_journals (id, transaction_type, financial_event_type, source_type, source_id, idempotency_key, idempotency_fingerprint, currency, description, status, created_at, is_reversed)
		VALUES (?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?, 'key_tr1', 'fp_tr1', 'INR', 'May sale', 'DRAFT', '2026-05-15 10:05:00+00', FALSE)
	`, journalID, payID.String()).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
		VALUES 
			(?, ?, ?, 'DEBIT', 1000, 'INR', '2026-05-15 10:05:00+00'),
			(?, ?, ?, 'CREDIT', 1000, 'INR', '2026-05-15 10:05:00+00')
	`, uuid.New(), journalID, acc1010, uuid.New(), journalID, acc4010).Error)
	require.NoError(t, db.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = '2026-05-15 10:05:00+00' WHERE id = ?`, journalID).Error)

	// Seed refund in 2026-06
	refundID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, amount, status, reason, created_at, processed_at)
		VALUES (?, ?, ?, 200, 'SUCCESS', 'CUSTOMER_CANCELLATION', '2026-06-10 12:00:00+00', '2026-06-10 12:00:00+00')
	`, refundID, bookingID, payID).Error)

	// Query trends spanning 2026-05 to 2026-06
	fromTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)

	trends, err := repo.GetTrends(ctx, domain.AdminDashboardFilter{
		From: &fromTime,
		To:   &toTime,
	})
	require.NoError(t, err)

	require.Len(t, trends.RevenueTrends, 2)
	assert.Equal(t, "2026-05", trends.RevenueTrends[0].Period)
	assert.True(t, decimal.NewFromInt(1000).Equal(trends.RevenueTrends[0].GrossVolume))
	assert.True(t, decimal.NewFromInt(1000).Equal(trends.RevenueTrends[0].PlatformFees))

	assert.Equal(t, "2026-06", trends.RevenueTrends[1].Period)
	assert.True(t, trends.RevenueTrends[1].GrossVolume.IsZero())

	require.Len(t, trends.BookingTrends, 2)
	assert.Equal(t, "2026-05", trends.BookingTrends[0].Period)
	assert.Equal(t, int64(1), trends.BookingTrends[0].BookingCount)
	assert.Equal(t, int64(2), trends.BookingTrends[0].TicketsSold)

	require.Len(t, trends.RefundTrends, 2)
	assert.Equal(t, "2026-06", trends.RefundTrends[1].Period)
	assert.True(t, decimal.NewFromInt(200).Equal(trends.RefundTrends[1].RefundAmount))
	assert.Equal(t, int64(1), trends.RefundTrends[1].RefundCount)

	// Date-filtered Financial Overview: Only 2026-05 (Gross captured 1000, Refunds 0)
	mayFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	mayTo := time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC)
	foMay, err := repo.GetFinancialOverview(ctx, domain.AdminDashboardFilter{
		From: &mayFrom,
		To:   &mayTo,
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1000).Equal(foMay.GrossCapturedAmount))
	assert.True(t, foMay.TotalRefunds.IsZero(), "Refund happened in June, so May refunds must be 0")
}

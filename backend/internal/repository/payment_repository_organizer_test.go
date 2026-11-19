package repository_test

import (
	"context"
	"fmt"
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

func setupOrganizerPaymentTestTables(t *testing.T, db *gorm.DB) {
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

func seedAccountedJournal(
	t *testing.T,
	db *gorm.DB,
	paymentID string,
	amount decimal.Decimal,
	subtotal decimal.Decimal,
	fee decimal.Decimal,
	orgID uuid.UUID,
	eventID uuid.UUID,
	acc1010, acc2010, acc4010 uuid.UUID,
) uuid.UUID {
	t.Helper()
	journalID := uuid.New()
	fingerprint := fmt.Sprintf("%064x", 123)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO finance_journals (
				id, transaction_type, financial_event_type, source_type, source_id,
				idempotency_key, idempotency_fingerprint, currency, description,
				status, created_at, is_reversed
			) VALUES (
				?, 'PAYMENT', 'TICKET_SALE_CAPTURED', 'PAYMENT', ?,
				?, ?, 'INR', 'Payment journal',
				'DRAFT', NOW(), FALSE
			)
		`, journalID, paymentID, "key_"+paymentID, fingerprint).Error; err != nil {
			return err
		}

		// Leg 1: Debit Razorpay Clearing (1010)
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, created_at)
			VALUES (?, ?, ?, 'DEBIT', ?, 'INR', NOW())
		`, uuid.New(), journalID, acc1010, amount).Error; err != nil {
			return err
		}

		// Leg 2: Credit Organizer Payable (2010)
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, organizer_id, event_id, created_at)
			VALUES (?, ?, ?, 'CREDIT', ?, 'INR', ?, ?, NOW())
		`, uuid.New(), journalID, acc2010, subtotal, orgID, eventID).Error; err != nil {
			return err
		}

		// Leg 3: Credit Platform Fee Revenue (4010)
		if err := tx.Exec(`
			INSERT INTO finance_ledger_entries (id, journal_id, account_id, entry_type, amount, currency, event_id, created_at)
			VALUES (?, ?, ?, 'CREDIT', ?, 'INR', ?, NOW())
		`, uuid.New(), journalID, acc4010, fee, eventID).Error; err != nil {
			return err
		}

		// Post the balanced journal
		return tx.Exec(`UPDATE finance_journals SET status = 'POSTED', posted_at = NOW() WHERE id = ?`, journalID).Error
	})
	require.NoError(t, err)

	return journalID
}

func TestPaymentRepository_OrganizerPaymentVisibility_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupOrganizerPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, acc2010, acc4010, _ := setupAdminFinanceAccounts(t, db)

	// -------------------------------------------------------------------------
	// 1. Seed Organizer A & Organizer B
	// -------------------------------------------------------------------------
	userA := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org User A', 'orga@example.com')`, userA).Error)
	orgA := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgA, userA).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Alpha Productions')`, uuid.New(), orgA).Error)

	userB := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org User B', 'orgb@example.com')`, userB).Error)
	orgB := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgB, userB).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Beta Events')`, uuid.New(), orgB).Error)

	// Customers
	cust1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Customer One', 'cust1@example.com')`, cust1).Error)
	cust2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Customer Two', 'cust2@example.com')`, cust2).Error)

	// -------------------------------------------------------------------------
	// 2. Events for Organizer A
	// -------------------------------------------------------------------------
	catID := uuid.New()
	eventA1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Rock Fest 2026')`, eventA1, orgA, catID).Error)

	eventA2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Jazz Night 2026')`, eventA2, orgA, catID).Error)

	// Event for Organizer B
	eventB1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Secret Beta Gala')`, eventB1, orgB, catID).Error)

	// -------------------------------------------------------------------------
	// 3. Seed Transactions for Event A1 (Multiple payments, multiple booking items, multiple attempts)
	// -------------------------------------------------------------------------
	now := time.Now().UTC()
	paidAt1 := now.Add(-2 * time.Hour)

	// Booking A1_1: 2 items (2 Gen + 1 VIP = 3 tickets). Subtotal 900, Fee 100, Total 1000. CONFIRMED.
	bA1_1 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-A1-001', ?, ?, 900.00, 100.00, 0.00, 0.00, 1000.00, 'CONFIRMED', ?)
	`, bA1_1, cust1, eventA1, paidAt1).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 2, 300.00, 600.00), (?, ?, 1, 300.00, 300.00)
	`, uuid.New(), bA1_1, uuid.New(), bA1_1).Error)

	// Payment A1_1_1: FAILED attempt
	pA1_1_failed := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, provider, amount, status, created_at, updated_at)
		VALUES (?::uuid, ?, 'RAZORPAY', 1000.00, 'FAILED', ?, ?)
	`, pA1_1_failed, bA1_1, paidAt1.Add(-5*time.Minute), paidAt1.Add(-5*time.Minute)).Error)

	// Payment A1_1_2: SUCCESS attempt (Accounted via journal)
	pA1_1_success := uuid.New().String()
	txA1 := "pay_A1_SUCCESS_1"
	ordA1 := "order_A1_1"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, ?, 'RAZORPAY', ?, 'UPI', 1000.00, 'SUCCESS', ?, ?, ?)
	`, pA1_1_success, bA1_1, txA1, ordA1, paidAt1, paidAt1, paidAt1).Error)
	seedAccountedJournal(t, db, pA1_1_success, decimal.NewFromInt(1000), decimal.NewFromInt(900), decimal.NewFromInt(100), orgA, eventA1, acc1010, acc2010, acc4010)

	// Booking A1_2: Free booking (Total 0, Subtotal 0, Fee 0, 2 tickets)
	bA1_free := uuid.New()
	paidAtFree := now.Add(-1 * time.Hour)
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-A1-FREE', ?, ?, 0.00, 0.00, 0.00, 0.00, 0.00, 'CONFIRMED', ?)
	`, bA1_free, cust2, eventA1, paidAtFree).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 2, 0.00, 0.00)
	`, uuid.New(), bA1_free).Error)

	pA1_free := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, provider, payment_method, amount, status, created_at, updated_at)
		VALUES (?::uuid, ?, 'RAZORPAY', 'FREE', 0.00, 'SUCCESS', ?, ?)
	`, pA1_free, bA1_free, paidAtFree, paidAtFree).Error)

	// Booking A1_3: Captured but UNALLOCATED payment (e.g. reservation expired after capture)
	bA1_unalloc := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-A1-EXPIRED', ?, ?, 450.00, 50.00, 0.00, 0.00, 500.00, 'EXPIRED', ?)
	`, bA1_unalloc, cust1, eventA1, now).Error)
	pA1_unalloc := uuid.New().String()
	txUnalloc := "pay_A1_UNALLOC"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, ?, 'RAZORPAY', 'CARD', 500.00, 'REFUND_REQUIRED', ?, ?, ?)
	`, pA1_unalloc, bA1_unalloc, txUnalloc, now, now, now).Error)

	// Booking A1_4: Pending payment attempt
	bA1_pending := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-A1-PENDING', ?, ?, 200.00, 20.00, 0.00, 0.00, 220.00, 'PENDING', ?)
	`, bA1_pending, cust2, eventA1, now).Error)
	pA1_pending := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, provider, amount, status, created_at, updated_at)
		VALUES (?::uuid, ?, 'RAZORPAY', 220.00, 'PENDING', ?, ?)
	`, pA1_pending, bA1_pending, now, now).Error)

	// -------------------------------------------------------------------------
	// 4. Seed Transactions for Event A2 (Second event for Org A)
	// -------------------------------------------------------------------------
	// Booking A2_1: 1 ticket, subtotal 300, fee 30, total 330. CONFIRMED.
	bA2_1 := uuid.New()
	paidAtA2 := now.Add(-30 * time.Minute)
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-A2-001', ?, ?, 300.00, 30.00, 0.00, 0.00, 330.00, 'CONFIRMED', ?)
	`, bA2_1, cust1, eventA2, paidAtA2).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 1, 300.00, 300.00)
	`, uuid.New(), bA2_1).Error)

	pA2_1 := uuid.New().String()
	txA2 := "pay_A2_SUCCESS"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, ?, 'RAZORPAY', 'CARD', 330.00, 'SUCCESS', ?, ?, ?)
	`, pA2_1, bA2_1, txA2, paidAtA2, paidAtA2, paidAtA2).Error)
	seedAccountedJournal(t, db, pA2_1, decimal.NewFromInt(330), decimal.NewFromInt(300), decimal.NewFromInt(30), orgA, eventA2, acc1010, acc2010, acc4010)

	// -------------------------------------------------------------------------
	// 5. Seed Transactions for Event B1 (Belonging to Organizer B)
	// -------------------------------------------------------------------------
	bB1_1 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-B1-SECRET', ?, ?, 5000.00, 500.00, 0.00, 0.00, 5500.00, 'CONFIRMED', ?)
	`, bB1_1, cust2, eventB1, now).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 10, 500.00, 5000.00)
	`, uuid.New(), bB1_1).Error)

	pB1_1 := uuid.New().String()
	txB1 := "pay_B1_SECRET"
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, ?, 'RAZORPAY', 'CARD', 5500.00, 'SUCCESS', ?, ?, ?)
	`, pB1_1, bB1_1, txB1, now, now, now).Error)
	seedAccountedJournal(t, db, pB1_1, decimal.NewFromInt(5500), decimal.NewFromInt(5000), decimal.NewFromInt(500), orgB, eventB1, acc1010, acc2010, acc4010)

	// =========================================================================
	// TESTS
	// =========================================================================

	t.Run("Security: Organizer A cannot see Organizer B payments", func(t *testing.T) {
		res, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		// Organizer A has 6 payments total across its events (5 on A1, 1 on A2)
		assert.Equal(t, int64(6), res.TotalRecords)
		for _, p := range res.Payments {
			assert.NotEqual(t, pB1_1, p.ID, "Organizer A must not see Organizer B payment")
			assert.NotEqual(t, "BK-B1-SECRET", p.BookingNumber)
			assert.NotEqual(t, eventB1, p.Event.ID)
		}
	})

	t.Run("Security: Organizer A cannot see Organizer B event summaries", func(t *testing.T) {
		res, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		// Org A has 2 events
		assert.Equal(t, int64(2), res.TotalRecords)
		assert.Equal(t, int64(2), res.Overview.TotalEvents)
		for _, s := range res.Summaries {
			assert.NotEqual(t, eventB1, s.EventID)
			assert.NotEqual(t, "Secret Beta Gala", s.EventTitle)
		}
	})

	t.Run("Security: Cross-organizer search or event filter returns zero records", func(t *testing.T) {
		// Org A tries to search for Org B's secret booking
		resSearch, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Search:      "SECRET",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), resSearch.TotalRecords)
		assert.Empty(t, resSearch.Payments)

		// Org A tries to filter explicitly by Org B's event ID
		resEvent, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			EventID:     &eventB1,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), resEvent.TotalRecords)
		assert.Empty(t, resEvent.Payments)
	})

	t.Run("Validation: Nil organizer ID is rejected immediately", func(t *testing.T) {
		_, errList := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: uuid.Nil,
		})
		assert.ErrorIs(t, errList, domain.ErrInvalidOrganizerID)

		_, errSummary := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: uuid.Nil,
		})
		assert.ErrorIs(t, errSummary, domain.ErrInvalidOrganizerID)

		_, errOverview := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: uuid.Nil,
		})
		assert.ErrorIs(t, errOverview, domain.ErrInvalidOrganizerID)
	})

	t.Run("Multiple payment attempts and ticket quantity on listing", func(t *testing.T) {
		res, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			EventID:     &eventA1,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), res.TotalRecords)

		// Find the successful payment for Booking A1_1
		var successPayment *domain.OrganizerPaymentListItem
		var failedPayment *domain.OrganizerPaymentListItem
		var freePayment *domain.OrganizerPaymentListItem
		var unallocPayment *domain.OrganizerPaymentListItem

		for i := range res.Payments {
			p := &res.Payments[i]
			switch p.ID {
			case pA1_1_success:
				successPayment = p
			case pA1_1_failed:
				failedPayment = p
			case pA1_free:
				freePayment = p
			case pA1_unalloc:
				unallocPayment = p
			}
		}

		require.NotNil(t, successPayment)
		require.NotNil(t, failedPayment)
		require.NotNil(t, freePayment)
		require.NotNil(t, unallocPayment)

		// Successful payment checks:
		assert.Equal(t, "BK-A1-001", successPayment.BookingNumber)
		assert.Equal(t, "Customer One", successPayment.Customer.Name)
		assert.Equal(t, "cust1@example.com", successPayment.Customer.Email)
		assert.Equal(t, 3, successPayment.TicketQuantity, "Booking items: 2 + 1 = 3 tickets")
		assert.True(t, decimal.NewFromInt(1000).Equal(successPayment.Amount))
		assert.Equal(t, domain.PaymentStatusSuccess, successPayment.Status)
		assert.True(t, successPayment.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusAccounted, successPayment.AllocationStatus)
		assert.Equal(t, domain.PaymentMethodUPI, successPayment.Method)
		assert.Equal(t, txA1, *successPayment.TransactionID)

		// Failed payment checks:
		assert.Equal(t, "BK-A1-001", failedPayment.BookingNumber)
		assert.Equal(t, 3, failedPayment.TicketQuantity)
		assert.Equal(t, domain.PaymentStatusFailed, failedPayment.Status)
		assert.False(t, failedPayment.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusNotApplicable, failedPayment.AllocationStatus)

		// Free booking checks:
		assert.Equal(t, "BK-A1-FREE", freePayment.BookingNumber)
		assert.Equal(t, 2, freePayment.TicketQuantity)
		assert.True(t, decimal.Zero.Equal(freePayment.Amount))
		assert.Equal(t, domain.PaymentStatusSuccess, freePayment.Status)
		assert.Equal(t, domain.AllocationStatusAccounted, freePayment.AllocationStatus, "Free booking confirmed exception must be ACCOUNTED")

		// Unallocated payment checks:
		assert.Equal(t, "BK-A1-EXPIRED", unallocPayment.BookingNumber)
		assert.Equal(t, domain.PaymentStatusRefundRequired, unallocPayment.Status)
		assert.True(t, unallocPayment.HasGatewayCapture)
		assert.Equal(t, domain.AllocationStatusUnallocated, unallocPayment.AllocationStatus)
	})

	t.Run("Event-wise Summary: Aggregation without row multiplication", func(t *testing.T) {
		res, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Summaries, 2)

		// Summaries are ordered by gross_captured_amount DESC:
		// Event A1 gross captured: 1000 (P_success) + 0 (P_free) + 500 (P_unalloc) = 1500
		// Event A2 gross captured: 330
		sA1 := res.Summaries[0]
		sA2 := res.Summaries[1]

		assert.Equal(t, eventA1, sA1.EventID)
		assert.Equal(t, "Rock Fest 2026", sA1.EventTitle)
		assert.Equal(t, int64(5), sA1.TotalPaymentsCount, "Failed, success, free, unallocated, pending payments")
		assert.Equal(t, int64(2), sA1.SuccessfullyCompletedPaymentsCount, "P_success and P_free")
		assert.True(t, decimal.NewFromInt(1500).Equal(sA1.GrossCapturedAmount), "1000 + 500 captured = 1500")
		assert.True(t, decimal.NewFromInt(100).Equal(sA1.PlatformFees), "Fee on confirmed completed sales")
		assert.True(t, decimal.NewFromInt(900).Equal(sA1.OrganizerEarnings), "Subtotal on confirmed completed sales")
		assert.Equal(t, int64(5), sA1.TicketsSold, "3 tickets (bA1_1) + 2 tickets (bA1_free) = 5 tickets")
		assert.Equal(t, int64(1), sA1.FailedPaymentsCount)
		assert.Equal(t, int64(1), sA1.RefundRequiredCount)
		assert.Equal(t, int64(1), sA1.UnallocatedCapturedCount)

		// Event A2 checks:
		assert.Equal(t, eventA2, sA2.EventID)
		assert.Equal(t, "Jazz Night 2026", sA2.EventTitle)
		assert.Equal(t, int64(1), sA2.TotalPaymentsCount)
		assert.Equal(t, int64(1), sA2.SuccessfullyCompletedPaymentsCount)
		assert.True(t, decimal.NewFromInt(330).Equal(sA2.GrossCapturedAmount))
		assert.True(t, decimal.NewFromInt(30).Equal(sA2.PlatformFees))
		assert.True(t, decimal.NewFromInt(300).Equal(sA2.OrganizerEarnings))
		assert.Equal(t, int64(1), sA2.TicketsSold)
	})

	t.Run("Overview: Metrics independent of pagination", func(t *testing.T) {
		// Request page 1 with limit 1
		res, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Page:        1,
			Limit:       1,
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		// Only 1 event on this page
		assert.Len(t, res.Summaries, 1)
		assert.Equal(t, int64(2), res.TotalRecords)
		assert.Equal(t, 2, res.TotalPages)

		// Overview MUST reflect totals across ALL matching events (A1 + A2)
		ov := res.Overview
		assert.Equal(t, int64(2), ov.TotalEvents)
		assert.Equal(t, int64(6), ov.TotalPaymentsCount)
		assert.Equal(t, int64(3), ov.SuccessfullyCompletedPaymentsCount)

		// Gross captured: 1500 (A1) + 330 (A2) = 1830
		assert.True(t, decimal.NewFromInt(1830).Equal(ov.GrossCapturedAmount))

		// Tickets sold: 5 (A1) + 1 (A2) = 6
		assert.Equal(t, int64(6), ov.TotalTicketsSold)

		// Platform fees: 100 (A1) + 30 (A2) = 130
		assert.True(t, decimal.NewFromInt(130).Equal(ov.TotalPlatformFees))

		// Organizer earnings: 900 (A1) + 300 (A2) = 1200
		assert.True(t, decimal.NewFromInt(1200).Equal(ov.OrganizerEarnings))
		assert.True(t, decimal.NewFromInt(1200).Equal(ov.TotalTicketSalesAmount))

		assert.Equal(t, int64(1), ov.RefundRequiredCount)
		assert.Equal(t, int64(1), ov.UnallocatedCapturedCount)

		// Direct call to GetOrganizerPaymentOverview produces the exact same overview
		directOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
		})
		require.NoError(t, err)
		assert.Equal(t, ov, *directOverview)
	})

	t.Run("Filtering by AllocationStatus and PaymentStatus", func(t *testing.T) {
		// Filter allocation_status = ACCOUNTED
		resAccounted, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID:      orgA,
			AllocationStatus: "ACCOUNTED",
		})
		require.NoError(t, err)
		// Accounted: pA1_1_success, pA1_free, pA2_1 = 3 payments
		assert.Equal(t, int64(3), resAccounted.TotalRecords)
		for _, p := range resAccounted.Payments {
			assert.Equal(t, domain.AllocationStatusAccounted, p.AllocationStatus)
		}

		// Filter allocation_status = UNALLOCATED
		resUnalloc, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID:      orgA,
			AllocationStatus: "UNALLOCATED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resUnalloc.TotalRecords)
		assert.Equal(t, pA1_unalloc, resUnalloc.Payments[0].ID)

		// Filter status = FAILED
		resFailed, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Status:      "FAILED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resFailed.TotalRecords)
		assert.Equal(t, pA1_1_failed, resFailed.Payments[0].ID)
	})

	t.Run("Search and Date filtering", func(t *testing.T) {
		// Search by transaction ID
		resTx, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Search:      "A2_SUCCESS",
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), resTx.TotalRecords)
		assert.Equal(t, pA2_1, resTx.Payments[0].ID)

		// Search by customer email
		resCust, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Search:      "cust1@example.com",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), resCust.TotalRecords)

		// Date filter: exclude the oldest payment
		fromDate := paidAt1.Add(10 * time.Minute)
		resDate, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			From:        &fromDate,
		})
		require.NoError(t, err)
		// Should exclude pA1_1_failed and pA1_1_success (created at paidAt1)
		for _, p := range resDate.Payments {
			assert.NotEqual(t, pA1_1_success, p.ID)
			assert.NotEqual(t, pA1_1_failed, p.ID)
		}
	})

	t.Run("Pagination boundaries and offset overflow safety", func(t *testing.T) {
		// Requesting page beyond total records returns empty list
		resEmpty, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Page:        999,
			Limit:       10,
		})
		require.NoError(t, err)
		assert.Empty(t, resEmpty.Payments)
		assert.Equal(t, int64(6), resEmpty.TotalRecords)
		assert.Equal(t, 999, resEmpty.Page)

		// Large page number does not panic or crash
		resHuge, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Page:        1_000_000,
			Limit:       100,
		})
		require.NoError(t, err)
		assert.Empty(t, resHuge.Payments)
	})

	t.Run("Status and Allocation filter consistency across Summaries and Overview", func(t *testing.T) {
		// 1. Filter by Status = FAILED
		resFailedSum, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Status:      "FAILED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resFailedSum.Overview.TotalEvents, "Only 1 event has failed payments")
		assert.Equal(t, int64(1), resFailedSum.Overview.TotalPaymentsCount)
		assert.Equal(t, int64(0), resFailedSum.Overview.SuccessfullyCompletedPaymentsCount)
		assert.True(t, decimal.Zero.Equal(resFailedSum.Overview.GrossCapturedAmount))
		assert.Equal(t, int64(0), resFailedSum.Overview.TotalTicketsSold)
		require.Len(t, resFailedSum.Summaries, 1)
		assert.Equal(t, eventA1, resFailedSum.Summaries[0].EventID)
		assert.Equal(t, int64(1), resFailedSum.Summaries[0].TotalPaymentsCount)
		assert.Equal(t, int64(1), resFailedSum.Summaries[0].FailedPaymentsCount)
		assert.Equal(t, int64(0), resFailedSum.Summaries[0].TicketsSold)

		// Overview direct call consistency
		failedOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Status:      "FAILED",
		})
		require.NoError(t, err)
		assert.Equal(t, resFailedSum.Overview, *failedOverview)

		// 2. Filter by Status = SUCCESS
		resSuccessSum, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			Status:      "SUCCESS",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), resSuccessSum.Overview.TotalEvents)
		assert.Equal(t, int64(3), resSuccessSum.Overview.TotalPaymentsCount)
		assert.Equal(t, int64(3), resSuccessSum.Overview.SuccessfullyCompletedPaymentsCount)
		// Gross captured of successful payments: 1000 + 0 + 330 = 1330
		assert.True(t, decimal.NewFromInt(1330).Equal(resSuccessSum.Overview.GrossCapturedAmount))
		assert.Equal(t, int64(6), resSuccessSum.Overview.TotalTicketsSold)
		require.Len(t, resSuccessSum.Summaries, 2)

		// 3. Filter by AllocationStatus = UNALLOCATED
		resUnallocSum, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID:      orgA,
			AllocationStatus: "UNALLOCATED",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resUnallocSum.Overview.TotalEvents)
		assert.Equal(t, int64(1), resUnallocSum.Overview.UnallocatedCapturedCount)
		assert.True(t, decimal.NewFromInt(500).Equal(resUnallocSum.Overview.GrossCapturedAmount))
		require.Len(t, resUnallocSum.Summaries, 1)
		assert.Equal(t, eventA1, resUnallocSum.Summaries[0].EventID)
		assert.Equal(t, int64(1), resUnallocSum.Summaries[0].UnallocatedCapturedCount)
	})

	t.Run("Date range boundaries: calendar dates vs timestamps", func(t *testing.T) {
		// When a calendar date is passed as 'To' (with 00:00:00 time), normalizeDateRange includes the entire day
		calDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		resCal, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			To:          &calDate,
		})
		require.NoError(t, err)
		// All 6 payments were created today, so calendar date 'today' includes all of them
		assert.Equal(t, int64(6), resCal.TotalRecords)

		// When yesterday's calendar date is used, today's payments are excluded
		yesterday := calDate.AddDate(0, 0, -1)
		resYesterday, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			To:          &yesterday,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), resYesterday.TotalRecords)

		// When explicit timestamps with hour/minute/second precision are used, precision is preserved
		cutoff := now.Add(-10 * time.Minute)
		resPrecise, err := repo.FindOrganizerPayments(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
			To:          &cutoff,
		})
		require.NoError(t, err)
		// Payments created before cutoff: paidAt1 (2 payments) and paidAtA2 (1 payment) and paidAtFree (1 payment) = 4
		assert.Equal(t, int64(4), resPrecise.TotalRecords)
		for _, p := range resPrecise.Payments {
			assert.True(t, p.CreatedAt.Before(cutoff) || p.CreatedAt.Equal(cutoff))
		}
	})

	t.Run("Event summary totals match overview totals exactly", func(t *testing.T) {
		res, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
			OrganizerID: orgA,
		})
		require.NoError(t, err)

		var sumTickets int64
		var sumGross decimal.Decimal
		var sumPlatformFee decimal.Decimal
		var sumEarnings decimal.Decimal
		var sumPaymentsCount int64
		var sumSuccessCount int64

		for _, s := range res.Summaries {
			sumTickets += s.TicketsSold
			sumGross = sumGross.Add(s.GrossCapturedAmount)
			sumPlatformFee = sumPlatformFee.Add(s.PlatformFees)
			sumEarnings = sumEarnings.Add(s.OrganizerEarnings)
			sumPaymentsCount += s.TotalPaymentsCount
			sumSuccessCount += s.SuccessfullyCompletedPaymentsCount
		}

		assert.Equal(t, res.Overview.TotalTicketsSold, sumTickets)
		assert.True(t, res.Overview.GrossCapturedAmount.Equal(sumGross))
		assert.True(t, res.Overview.TotalPlatformFees.Equal(sumPlatformFee))
		assert.True(t, res.Overview.OrganizerEarnings.Equal(sumEarnings))
		assert.Equal(t, res.Overview.TotalPaymentsCount, sumPaymentsCount)
		assert.Equal(t, res.Overview.SuccessfullyCompletedPaymentsCount, sumSuccessCount)
		assert.Equal(t, res.Overview.TotalEvents, int64(len(res.Summaries)))
	})
}

func TestPaymentRepository_DuplicateSuccessfulPaymentsProtection_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupOrganizerPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org User', 'org_dup@example.com')`, userID).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id, status) VALUES (?, ?, 'APPROVED')`, orgID, userID).Error)

	custID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Dup Customer', 'cust_dup@example.com')`, custID).Error)

	catID := uuid.New()
	eventID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Duplicate Payment Test Event')`, eventID, orgID, catID).Error)

	now := time.Now().UTC()
	// Booking: 3 tickets, subtotal 600, platform fee 60, total 660. CONFIRMED.
	bookingID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-DUP-001', ?, ?, 600.00, 60.00, 0.00, 0.00, 660.00, 'CONFIRMED', ?)
	`, bookingID, custID, eventID, now).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 2, 200.00, 400.00), (?, ?, 1, 200.00, 200.00)
	`, uuid.New(), bookingID, uuid.New(), bookingID).Error)

	// Payment 1: SUCCESS attempt 1
	p1 := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, 'tx_dup_1', 'RAZORPAY', 'UPI', 660.00, 'SUCCESS', ?, ?, ?)
	`, p1, bookingID, now.Add(-5*time.Minute), now.Add(-5*time.Minute), now.Add(-5*time.Minute)).Error)

	// Payment 2: Duplicate SUCCESS attempt 2 for the exact same booking
	p2 := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, 'tx_dup_2', 'RAZORPAY', 'CARD', 660.00, 'SUCCESS', ?, ?, ?)
	`, p2, bookingID, now, now, now).Error)

	// Query Event Summaries & Overview
	res, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{
		OrganizerID: orgID,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Summaries, 1)

	s := res.Summaries[0]
	ov := res.Overview

	// CRITICAL ASSERTIONS:
	// 1. Payment counts: 2 total payments, 2 successful payments
	assert.Equal(t, int64(2), s.TotalPaymentsCount)
	assert.Equal(t, int64(2), s.SuccessfullyCompletedPaymentsCount)
	assert.Equal(t, int64(2), ov.TotalPaymentsCount)
	assert.Equal(t, int64(2), ov.SuccessfullyCompletedPaymentsCount)

	// 2. Gross captured amount: sums both gateway captures = 660 + 660 = 1320
	assert.True(t, decimal.NewFromInt(1320).Equal(s.GrossCapturedAmount), "Gross captured must reflect both captured payments")
	assert.True(t, decimal.NewFromInt(1320).Equal(ov.GrossCapturedAmount))

	// 3. BOOKING-LEVEL METRICS MUST NOT BE INFLATED:
	// Tickets sold must be 3 (NOT 6!)
	assert.Equal(t, int64(3), s.TicketsSold, "Tickets sold must not be multiplied by duplicate successful payments")
	assert.Equal(t, int64(3), ov.TotalTicketsSold)

	// Platform fees must be 60 (NOT 120!)
	assert.True(t, decimal.NewFromInt(60).Equal(s.PlatformFees), "Platform fees must not be multiplied by duplicate successful payments")
	assert.True(t, decimal.NewFromInt(60).Equal(ov.TotalPlatformFees))

	// Organizer earnings must be 600 (NOT 1200!)
	assert.True(t, decimal.NewFromInt(600).Equal(s.OrganizerEarnings), "Organizer earnings must not be multiplied by duplicate successful payments")
	assert.True(t, decimal.NewFromInt(600).Equal(ov.OrganizerEarnings))
	assert.True(t, decimal.NewFromInt(600).Equal(ov.TotalTicketSalesAmount))
}

func TestPaymentRepository_OrganizerEarnings_DeductsSuccessfulRefunds_Postgres(t *testing.T) {
	db := initTestDatabase(t)
	setupOrganizerPaymentTestTables(t, db)
	repo := repository.NewPaymentRepository(db, slog.Default())
	ctx := context.Background()

	acc1010, acc2010, acc4010, _ := setupAdminFinanceAccounts(t, db)

	userID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Org User Refund', 'orgrefund@example.com')`, userID).Error)
	orgID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO organizers (id, user_id) VALUES (?, ?)`, orgID, userID).Error)
	require.NoError(t, db.Exec(`INSERT INTO organizer_profiles (id, organizer_id, business_name) VALUES (?, ?, 'Refund Test Org')`, uuid.New(), orgID).Error)

	custID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, full_name, email) VALUES (?, 'Customer Refund', 'custrefund@example.com')`, custID).Error)

	catID := uuid.New()
	eventID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO events (id, organizer_id, category_id, title) VALUES (?, ?, ?, 'Refund Deduction Concert')`, eventID, orgID, catID).Error)

	now := time.Now().UTC()

	// Booking: Subtotal 900, Fee 100, Total 1000. CONFIRMED.
	bookingID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-REFUND-001', ?, ?, 900.00, 100.00, 0.00, 0.00, 1000.00, 'CONFIRMED', ?)
	`, bookingID, custID, eventID, now).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 3, 300.00, 900.00)
	`, uuid.New(), bookingID).Error)

	// Payment: SUCCESS for 1000
	paymentID := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, 'tx_refund_001', 'RAZORPAY', 'order_ref_1', 'UPI', 1000.00, 'SUCCESS', ?, ?, ?)
	`, paymentID, bookingID, now, now, now).Error)
	seedAccountedJournal(t, db, paymentID, decimal.NewFromInt(1000), decimal.NewFromInt(900), decimal.NewFromInt(100), orgID, eventID, acc1010, acc2010, acc4010)

	// Initial check before refund:
	// Earnings should be 900.00
	initOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{OrganizerID: orgID})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(900).Equal(initOverview.OrganizerEarnings))
	assert.True(t, decimal.NewFromInt(900).Equal(initOverview.TotalTicketSalesAmount))

	// Now insert a successful customer refund of 300.00 for one ticket
	refundID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, amount, status, reason, gateway_refund_id, processed_at, created_at, updated_at)
		VALUES (?, ?, ?::uuid, 300.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', 'rfnd_gw_300', ?, ?, ?)
	`, refundID, bookingID, paymentID, now, now, now).Error)

	// Post-refund check:
	// Earnings should now be 600.00 (900 - 300)
	postOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{OrganizerID: orgID})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(600).Equal(postOverview.OrganizerEarnings), "Organizer earnings must deduct successful refunds")
	assert.True(t, decimal.NewFromInt(600).Equal(postOverview.TotalTicketSalesAmount), "Ticket sales must deduct successful refunds")

	// Verify Event-wise Summary
	eventSummaries, err := repo.GetOrganizerEventPaymentSummaries(ctx, domain.OrganizerPaymentFilter{OrganizerID: orgID})
	require.NoError(t, err)
	require.Len(t, eventSummaries.Summaries, 1)
	assert.True(t, decimal.NewFromInt(600).Equal(eventSummaries.Summaries[0].OrganizerEarnings), "Event summary organizer earnings must deduct successful refunds")

	// -------------------------------------------------------------------------
	// Verification of No Double Deduction:
	// Insert a second booking for the same event that is CANCELLED (e.g., event cancellation or full cancellation).
	// Because b.status = 'CANCELLED', it is excluded from primarySuccessfulPaymentCondition.
	// Its refund must NOT be subtracted from the confirmed booking's earnings.
	// -------------------------------------------------------------------------
	cancelledBookingID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-REFUND-CANCELLED', ?, ?, 500.00, 50.00, 0.00, 0.00, 550.00, 'CANCELLED', ?)
	`, cancelledBookingID, custID, eventID, now).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 1, 500.00, 500.00)
	`, uuid.New(), cancelledBookingID).Error)

	cancelledPaymentID := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, 'tx_refund_cancelled', 'RAZORPAY', 'order_ref_canc', 'UPI', 550.00, 'SUCCESS', ?, ?, ?)
	`, cancelledPaymentID, cancelledBookingID, now, now, now).Error)

	// Refund for the cancelled booking
	cancelledRefundID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, amount, status, reason, gateway_refund_id, processed_at, created_at, updated_at)
		VALUES (?, ?, ?::uuid, 550.00, 'SUCCESS', 'EVENT_CANCELLATION', 'rfnd_gw_550', ?, ?, ?)
	`, cancelledRefundID, cancelledBookingID, cancelledPaymentID, now, now, now).Error)

	// Query overview again: earnings must remain exactly 600.00 (from the confirmed booking only)!
	postCancellationOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{OrganizerID: orgID})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(600).Equal(postCancellationOverview.OrganizerEarnings), "Cancelled booking refund must not be double deducted from organizer earnings")
	assert.True(t, decimal.NewFromInt(600).Equal(postCancellationOverview.TotalTicketSalesAmount), "Cancelled booking refund must not reduce confirmed ticket sales")

	// -------------------------------------------------------------------------
	// Verification of GREATEST(0, ...):
	// Even if refund equals or exceeds subtotal, earnings never go negative.
	// -------------------------------------------------------------------------
	zeroBookingID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bookings (id, booking_number, user_id, event_id, subtotal, platform_fee, tax, discount, total, status, created_at)
		VALUES (?, 'BK-REFUND-EXCESS', ?, ?, 200.00, 20.00, 0.00, 50.00, 170.00, 'CONFIRMED', ?)
	`, zeroBookingID, custID, eventID, now).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO booking_items (id, booking_id, quantity, unit_price, total_price)
		VALUES (?, ?, 1, 200.00, 200.00)
	`, uuid.New(), zeroBookingID).Error)
	zeroPaymentID := uuid.New().String()
	require.NoError(t, db.Exec(`
		INSERT INTO payments (id, booking_id, transaction_id, provider, provider_order_id, payment_method, amount, status, paid_at, created_at, updated_at)
		VALUES (?::uuid, ?, 'tx_refund_excess', 'RAZORPAY', 'order_ref_exc', 'UPI', 170.00, 'SUCCESS', ?, ?, ?)
	`, zeroPaymentID, zeroBookingID, now, now, now).Error)
	// Refund of 200 on subtotal-discount (200-50 = 150) -> net would be -50 without GREATEST
	excessRefundID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO refunds (id, booking_id, payment_id, amount, status, reason, gateway_refund_id, processed_at, created_at, updated_at)
		VALUES (?, ?, ?::uuid, 200.00, 'SUCCESS', 'CUSTOMER_CANCELLATION', 'rfnd_gw_excess', ?, ?, ?)
	`, excessRefundID, zeroBookingID, zeroPaymentID, now, now, now).Error)

	postExcessOverview, err := repo.GetOrganizerPaymentOverview(ctx, domain.OrganizerPaymentFilter{OrganizerID: orgID})
	require.NoError(t, err)
	// 600 from BK-REFUND-001 + GREATEST(0, 150-200) = 600 + 0 = 600!
	assert.True(t, decimal.NewFromInt(600).Equal(postExcessOverview.OrganizerEarnings), "Earnings must never become negative due to GREATEST protection")
}

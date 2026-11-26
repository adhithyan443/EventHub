package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type dummyConnPool struct{}

func (d *dummyConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}
func (d *dummyConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (d *dummyConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (d *dummyConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// TestComputeActiveTickets_AllCancelledOrRefunded verifies that when ticket records exist
// but all tickets are cancelled or refunded, the active ticket count is 0 and does not fall back
// to the original booking item quantity.
func TestComputeActiveTickets_AllCancelledOrRefunded(t *testing.T) {
	ticketRecordsExist := true
	activeValidOrUsedCount := int64(0)
	fallbackBookingQuantity := int64(4)

	result := computeActiveTickets(ticketRecordsExist, activeValidOrUsedCount, fallbackBookingQuantity)
	assert.Equal(t, int64(0), result, "when ticket records exist, count must reflect active tickets (0), not booking quantity")
}

// TestComputeActiveTickets_PartialValidOrUsed verifies that when some tickets are active and some cancelled,
// only the active (VALID or USED) tickets are counted.
func TestComputeActiveTickets_PartialValidOrUsed(t *testing.T) {
	ticketRecordsExist := true
	activeValidOrUsedCount := int64(2)
	fallbackBookingQuantity := int64(5)

	result := computeActiveTickets(ticketRecordsExist, activeValidOrUsedCount, fallbackBookingQuantity)
	assert.Equal(t, int64(2), result, "must count only VALID or USED tickets")
}

// TestComputeActiveTickets_NoTicketRecordsFallback verifies fallback to booking item quantity
// when no ticket records have been issued yet.
func TestComputeActiveTickets_NoTicketRecordsFallback(t *testing.T) {
	ticketRecordsExist := false
	activeValidOrUsedCount := int64(0)
	fallbackBookingQuantity := int64(3)

	result := computeActiveTickets(ticketRecordsExist, activeValidOrUsedCount, fallbackBookingQuantity)
	assert.Equal(t, int64(3), result, "when no ticket records exist, should fallback to booking items quantity")
}

// TestComputeOrganizerNetCaptured_NegativePreserved verifies that financial calculations
// preserve negative net captured amounts for observability and do not silently clamp to zero.
func TestComputeOrganizerNetCaptured_NegativePreserved(t *testing.T) {
	grossCaptured := decimal.NewFromInt(1000)
	totalRefunded := decimal.NewFromInt(1250) // More refunded than captured (e.g. gateway dispute / fee adjustment)

	netCaptured := computeOrganizerNetCaptured(grossCaptured, totalRefunded)
	expected := decimal.NewFromInt(-250)

	assert.True(t, netCaptured.Equal(expected), "expected net captured to be -250, got %s", netCaptured.String())
	assert.True(t, netCaptured.IsNegative(), "negative net captured must remain negative and not be clamped to zero")
}

// TestComputeAuthoritativeBookingCapture verifies that duplicate payment attempts for the same
// booking order do not multiply or double-count the authorized capture amount.
func TestComputeAuthoritativeBookingCapture(t *testing.T) {
	now := time.Now()

	t.Run("No payments", func(t *testing.T) {
		amount := computeAuthoritativeBookingCapture([]paymentSummaryRow{})
		assert.True(t, amount.IsZero())
	})

	t.Run("Only failed payment attempts", func(t *testing.T) {
		rows := []paymentSummaryRow{
			{Amount: decimal.NewFromInt(500), Status: "FAILED", PaidAt: nil},
		}
		amount := computeAuthoritativeBookingCapture(rows)
		assert.True(t, amount.IsZero())
	})

	t.Run("Single successful capture", func(t *testing.T) {
		rows := []paymentSummaryRow{
			{Amount: decimal.NewFromInt(1500), Status: "SUCCESS", PaidAt: &now},
		}
		amount := computeAuthoritativeBookingCapture(rows)
		assert.True(t, amount.Equal(decimal.NewFromInt(1500)))
	})

	t.Run("Duplicate successful attempts do not double-count", func(t *testing.T) {
		// Simulates a duplicate capture (e.g. webhook race + client confirmation)
		rows := []paymentSummaryRow{
			{Amount: decimal.NewFromInt(1000), Status: "SUCCESS", PaidAt: &now},
			{Amount: decimal.NewFromInt(1000), Status: "SUCCESS", PaidAt: &now},
		}
		amount := computeAuthoritativeBookingCapture(rows)
		assert.True(t, amount.Equal(decimal.NewFromInt(1000)), "duplicate captures must not inflate booking total to 2000")
	})
}

// TestResolveBookingRefundStatus verifies the authoritative refund status semantics.
func TestResolveBookingRefundStatus(t *testing.T) {
	gross := decimal.NewFromInt(1000)

	t.Run("No refund records", func(t *testing.T) {
		status := resolveBookingRefundStatus(gross, decimal.Zero, false, false, 0)
		assert.Equal(t, "", status, "empty string indicates no refund records")
	})

	t.Run("Pending refund does not get hidden by previous partial refund", func(t *testing.T) {
		// A partial refund of 300 succeeded previously, and another refund is currently PENDING.
		partialRefunded := decimal.NewFromInt(300)
		hasPendingRefund := true
		hasFailedRefund := false
		refundCount := 2

		status := resolveBookingRefundStatus(gross, partialRefunded, hasPendingRefund, hasFailedRefund, refundCount)
		assert.Equal(t, "PENDING", status, "pending refund must take precedence over past partial refund to alert organizer")
	})

	t.Run("Full refund achieved", func(t *testing.T) {
		fullRefunded := decimal.NewFromInt(1000)
		status := resolveBookingRefundStatus(gross, fullRefunded, false, false, 2)
		assert.Equal(t, "FULL", status)
	})

	t.Run("Partial refund completed with no pending", func(t *testing.T) {
		partialRefunded := decimal.NewFromInt(400)
		status := resolveBookingRefundStatus(gross, partialRefunded, false, false, 1)
		assert.Equal(t, "PARTIAL", status)
	})

	t.Run("Failed refund with no success or pending", func(t *testing.T) {
		status := resolveBookingRefundStatus(gross, decimal.Zero, false, true, 1)
		assert.Equal(t, "FAILED", status)
	})
}

// TestMatchesBookingRefundFilter verifies the filter predicate matching semantics
// for REFUNDED, FULL, and PARTIAL filters across all financial and refund scenarios.
func TestMatchesBookingRefundFilter(t *testing.T) {
	gross := decimal.NewFromInt(1000)

	t.Run("1. Partially refunded booking matches REFUNDED", func(t *testing.T) {
		partialRefund := decimal.NewFromInt(400)
		hasSuccessfulRefund := true
		hasPendingRefund := false
		hasFailedRefund := false

		matches := matchesBookingRefundFilter("REFUNDED", gross, partialRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.True(t, matches, "a partially refunded booking must match REFUNDED filter")
	})

	t.Run("2. Fully refunded booking matches REFUNDED", func(t *testing.T) {
		fullRefund := decimal.NewFromInt(1000)
		hasSuccessfulRefund := true
		hasPendingRefund := false
		hasFailedRefund := false

		matches := matchesBookingRefundFilter("REFUNDED", gross, fullRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.True(t, matches, "a fully refunded booking must match REFUNDED filter")
	})

	t.Run("3. Partially refunded booking matches PARTIAL but not FULL", func(t *testing.T) {
		partialRefund := decimal.NewFromInt(300)
		hasSuccessfulRefund := true
		hasPendingRefund := false
		hasFailedRefund := false

		matchesPartial := matchesBookingRefundFilter("PARTIAL", gross, partialRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.True(t, matchesPartial, "partially refunded booking must match PARTIAL filter")

		matchesFull := matchesBookingRefundFilter("FULL", gross, partialRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.False(t, matchesFull, "partially refunded booking must NOT match FULL filter")
	})

	t.Run("4. Fully refunded booking matches FULL", func(t *testing.T) {
		fullRefund := decimal.NewFromInt(1000)
		hasSuccessfulRefund := true
		hasPendingRefund := false
		hasFailedRefund := false

		matchesFull := matchesBookingRefundFilter("FULL", gross, fullRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.True(t, matchesFull, "fully refunded booking must match FULL filter")

		matchesPartial := matchesBookingRefundFilter("PARTIAL", gross, fullRefund, hasSuccessfulRefund, hasPendingRefund, hasFailedRefund)
		assert.False(t, matchesPartial, "fully refunded booking must NOT match PARTIAL filter")
	})

	t.Run("5. Booking without successful captured payment does not match FULL", func(t *testing.T) {
		zeroGross := decimal.Zero
		refundAmt := decimal.NewFromInt(500) // hypothetical edge case refund
		hasSuccessfulRefund := true

		matchesFull := matchesBookingRefundFilter("FULL", zeroGross, refundAmt, hasSuccessfulRefund, false, false)
		assert.False(t, matchesFull, "booking without successful capture must never match FULL filter")
	})

	t.Run("6. Pending and failed refund attempts alone do not match REFUNDED, PARTIAL, or FULL", func(t *testing.T) {
		zeroRefunded := decimal.Zero
		hasSuccessfulRefund := false

		// Scenario A: Pending refund attempt only
		assert.False(t, matchesBookingRefundFilter("REFUNDED", gross, zeroRefunded, hasSuccessfulRefund, true, false))
		assert.False(t, matchesBookingRefundFilter("PARTIAL", gross, zeroRefunded, hasSuccessfulRefund, true, false))
		assert.False(t, matchesBookingRefundFilter("FULL", gross, zeroRefunded, hasSuccessfulRefund, true, false))
		assert.True(t, matchesBookingRefundFilter("PENDING_REFUND", gross, zeroRefunded, hasSuccessfulRefund, true, false))

		// Scenario B: Failed refund attempt only
		assert.False(t, matchesBookingRefundFilter("REFUNDED", gross, zeroRefunded, hasSuccessfulRefund, false, true))
		assert.False(t, matchesBookingRefundFilter("PARTIAL", gross, zeroRefunded, hasSuccessfulRefund, false, true))
		assert.False(t, matchesBookingRefundFilter("FULL", gross, zeroRefunded, hasSuccessfulRefund, false, true))
		assert.False(t, matchesBookingRefundFilter("PENDING_REFUND", gross, zeroRefunded, hasSuccessfulRefund, false, true))
	})
}

// TestBuildOrganizerBookingBaseQuery_RefundFiltersSQL verifies that buildOrganizerBookingBaseQuery
// generates distinct and exact SQL predicates for REFUNDED, FULL, and PARTIAL.
func TestBuildOrganizerBookingBaseQuery_RefundFiltersSQL(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: &dummyConnPool{},
	}), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Skipf("skipping SQL generation test due to dialector initialization: %v", err)
		return
	}

	repo := &bookingRepository{db: db}
	ctx := context.Background()
	organizerID := uuid.New()

	t.Run("REFUNDED filter generates EXISTS predicate", func(t *testing.T) {
		filter := domain.OrganizerBookingFilter{
			OrganizerID:   organizerID,
			PaymentStatus: "REFUNDED",
		}
		stmt := repo.buildOrganizerBookingBaseQuery(ctx, filter).Session(&gorm.Session{DryRun: true}).Find(&[]organizerBookingRow{}).Statement
		sql := stmt.SQL.String()

		assert.True(t, strings.Contains(sql, "EXISTS ("), "REFUNDED must use EXISTS")
		assert.True(t, strings.Contains(sql, "r.status = 'SUCCESS'"), "REFUNDED must check for SUCCESS refunds")
		assert.False(t, strings.Contains(sql, "< ("), "REFUNDED must not require strictly less than captured")
	})

	t.Run("FULL filter generates full refund mathematical comparison", func(t *testing.T) {
		filter := domain.OrganizerBookingFilter{
			OrganizerID:   organizerID,
			PaymentStatus: "FULL",
		}
		stmt := repo.buildOrganizerBookingBaseQuery(ctx, filter).Session(&gorm.Session{DryRun: true}).Find(&[]organizerBookingRow{}).Statement
		sql := stmt.SQL.String()

		assert.True(t, strings.Contains(sql, ">= ("), "FULL must compare refunds >= payments")
		assert.True(t, strings.Contains(sql, "p.status = 'SUCCESS' AND p.paid_at IS NOT NULL"), "FULL must verify captured payments")
		assert.True(t, strings.Contains(sql, "> 0"), "FULL must ensure captured amount > 0")
	})

	t.Run("PARTIAL filter generates partial refund comparison", func(t *testing.T) {
		filter := domain.OrganizerBookingFilter{
			OrganizerID:   organizerID,
			PaymentStatus: "PARTIAL",
		}
		stmt := repo.buildOrganizerBookingBaseQuery(ctx, filter).Session(&gorm.Session{DryRun: true}).Find(&[]organizerBookingRow{}).Statement
		sql := stmt.SQL.String()

		assert.True(t, strings.Contains(sql, "< ("), "PARTIAL must compare refunds < payments")
		assert.True(t, strings.Contains(sql, "> 0"), "PARTIAL must ensure refund total > 0")
	})

	t.Run("PENDING_REFUND filter checks in-flight statuses", func(t *testing.T) {
		filter := domain.OrganizerBookingFilter{
			OrganizerID:   organizerID,
			PaymentStatus: "PENDING_REFUND",
		}
		stmt := repo.buildOrganizerBookingBaseQuery(ctx, filter).Session(&gorm.Session{DryRun: true}).Find(&[]organizerBookingRow{}).Statement
		sql := stmt.SQL.String()

		assert.True(t, strings.Contains(sql, "r.status IN ('PENDING', 'PROCESSING')"), "PENDING_REFUND must check PENDING and PROCESSING")
	})
}

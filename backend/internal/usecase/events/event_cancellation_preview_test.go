package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventCancellationPreview_EligiblePublishedEvent(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Booking 1: Confirmed with 2 tickets ($200 ticket, $10 fee, $5 tax = $215 total)
	b1ID := uuid.New()
	b1 := &domain.Booking{
		ID:            b1ID,
		UserID:        uuid.New(),
		EventID:       f.eventID,
		BookingNumber: "BK-PREV-1",
		Subtotal:      decimal.NewFromInt(200),
		PlatformFee:   decimal.NewFromInt(10),
		Tax:           decimal.NewFromInt(5),
		Total:         decimal.NewFromInt(215),
		Status:        domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[b1ID] = b1

	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(100), Status: domain.TicketStatusValid}
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(100), Status: domain.TicketStatusValid}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	f.txRepos.ticketRepo.tickets[t2.ID] = t2

	txID1 := "pay_test_prev_1"
	p1ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p1ID] = &domain.Payment{
		ID:            p1ID,
		BookingID:     b1ID.String(),
		Amount:        decimal.NewFromInt(215),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txID1,
	}

	// Active seat reservation hold
	resID := uuid.New()
	seatID := uuid.New()
	ttID := uuid.New()
	f.txRepos.reservationRepo.reservations[resID] = &domain.SeatReservation{
		ID:        resID,
		EventID:   f.eventID,
		UserID:    uuid.New(),
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	f.txRepos.reservationRepo.items[resID] = []domain.SeatReservationItem{
		{ID: uuid.New(), ReservationID: resID, SeatID: &seatID, Quantity: 1},
		{ID: uuid.New(), ReservationID: resID, TicketTypeID: &ttID, Quantity: 3},
	}
	f.txRepos.checkoutRepo.checkouts[resID] = &domain.Checkout{
		ID:            uuid.New(),
		ReservationID: resID,
		Status:        domain.CheckoutStatusActive,
	}

	preview, err := f.usecase.GetCancellationPreviewForOrganizer(ctx, f.organizerUserID, f.eventID)
	require.NoError(t, err)
	require.NotNil(t, preview)

	// Event details
	assert.Equal(t, f.eventID, preview.Event.ID)
	assert.Equal(t, f.event.Title, preview.Event.Title)
	assert.Equal(t, domain.EventStatusPublished, preview.Event.Status)
	assert.NotEmpty(t, preview.Event.ScheduledDate)

	// Eligibility
	assert.True(t, preview.Eligibility.IsCancellable)
	assert.Empty(t, preview.Eligibility.BlockingReasons)

	// Operational Impact
	assert.Equal(t, 1, preview.OperationalImpact.TotalBookings)
	assert.Equal(t, 1, preview.OperationalImpact.AffectedBookings)
	assert.Equal(t, 2, preview.OperationalImpact.TotalTicketsSold)
	assert.Equal(t, 2, preview.OperationalImpact.AffectedTickets)
	assert.Equal(t, 1, preview.OperationalImpact.ActiveReservationsReleased)
	assert.Equal(t, 1, preview.OperationalImpact.ReservedSeatsReleased)
	assert.Equal(t, 3, preview.OperationalImpact.GeneralAdmissionQuantityReleased)
	assert.Equal(t, 1, preview.OperationalImpact.ActiveCheckoutsExpired)

	// Financial Impact
	assert.Equal(t, "INR", preview.FinancialImpact.Currency)
	assert.True(t, preview.FinancialImpact.GrossCollectedAmount.Equal(decimal.NewFromInt(215)))
	assert.True(t, preview.FinancialImpact.AlreadyRefundedAmount.IsZero())
	assert.True(t, preview.FinancialImpact.ExpectedRefundAmount.Equal(decimal.NewFromInt(215)))
	assert.True(t, preview.FinancialImpact.PlatformFeeRefund.Equal(decimal.NewFromInt(10)))
	assert.True(t, preview.FinancialImpact.TaxRefund.Equal(decimal.NewFromInt(5)))
	assert.True(t, preview.FinancialImpact.OrganizerPayableRevocation.Equal(decimal.NewFromInt(200)))

	// Check that preview never mutates database
	assert.Equal(t, domain.EventStatusPublished, f.txRepos.eventRepo.events[f.eventID].Status)
	assert.Equal(t, domain.BookingStatusConfirmed, f.txRepos.bookingRepo.bookings[b1ID].Status)
	assert.Empty(t, f.txRepos.refundRepo.refunds)
}

func TestEventCancellationPreview_IneligibleReasons(t *testing.T) {
	ctx := context.Background()

	t.Run("DraftEvent", func(t *testing.T) {
		f := newCancellationFixture(t)
		f.txRepos.eventRepo.events[f.eventID].Status = domain.EventStatusDraft

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)
		assert.Contains(t, preview.Eligibility.BlockingReasons, "draft events cannot be cancelled; delete the draft instead")
	})

	t.Run("AlreadyCancelledEvent", func(t *testing.T) {
		f := newCancellationFixture(t)
		f.txRepos.eventRepo.events[f.eventID].Status = domain.EventStatusCancelled

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)
		assert.Contains(t, preview.Eligibility.BlockingReasons, "event is already cancelled")
	})

	t.Run("ConcludedEvent", func(t *testing.T) {
		f := newCancellationFixture(t)
		past := time.Now().Add(-48 * time.Hour)
		f.txRepos.scheduleRepo.schedules[f.eventID].EventDate = past
		f.txRepos.scheduleRepo.schedules[f.eventID].StartTime = past
		f.txRepos.scheduleRepo.schedules[f.eventID].EndTime = past.Add(2 * time.Hour)

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)
		assert.Contains(t, preview.Eligibility.BlockingReasons, "event has already concluded")
	})

	t.Run("CompletedSettlementBlocks", func(t *testing.T) {
		f := newCancellationFixture(t)
		sID := uuid.New()
		f.txRepos.settlementRepo.settlements[sID] = &domain.Settlement{
			ID:          sID,
			OrganizerID: f.organizerID,
			EventID:     &f.eventID,
			Amount:      decimal.NewFromInt(5000),
			Status:      domain.SettlementStatusCompleted,
		}

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)
		assert.True(t, preview.SettlementImpact.HasCompletedSettlement)
		assert.Contains(t, preview.Eligibility.BlockingReasons, "event has completed settlements with disbursed organizer funds")
	})

	t.Run("InFlightSettlementBlocks", func(t *testing.T) {
		f := newCancellationFixture(t)
		sID := uuid.New()
		f.txRepos.settlementRepo.settlements[sID] = &domain.Settlement{
			ID:          sID,
			OrganizerID: f.organizerID,
			EventID:     &f.eventID,
			Amount:      decimal.NewFromInt(5000),
			Status:      domain.SettlementStatusProcessing,
		}

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)
		assert.True(t, preview.SettlementImpact.HasInFlightSettlement)
		assert.Contains(t, preview.Eligibility.BlockingReasons, "event has approved or in-flight settlement payouts currently processing")
	})

	t.Run("PendingSettlementPermittedWithWarning", func(t *testing.T) {
		f := newCancellationFixture(t)
		sID := uuid.New()
		f.txRepos.settlementRepo.settlements[sID] = &domain.Settlement{
			ID:          sID,
			OrganizerID: f.organizerID,
			EventID:     &f.eventID,
			Amount:      decimal.NewFromInt(3500),
			Status:      domain.SettlementStatusPending,
		}

		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.True(t, preview.Eligibility.IsCancellable)
		assert.Empty(t, preview.Eligibility.BlockingReasons)
		assert.Equal(t, 1, preview.SettlementImpact.PendingSettlementsCount)
		assert.True(t, preview.SettlementImpact.PendingSettlementsTotal.Equal(decimal.NewFromInt(3500)))
		assert.Contains(t, preview.SettlementImpact.SettlementWarning, "automatically rejected upon cancellation")
	})
}

func TestEventCancellationPreview_PartialAndComplexRefunds(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Booking 1: Confirmed booking ($100 ticket + $10 fee + $5 tax = $115 total)
	b1ID := uuid.New()
	b1 := &domain.Booking{
		ID:            b1ID,
		UserID:        uuid.New(),
		EventID:       f.eventID,
		BookingNumber: "BK-CMPLX-1",
		Subtotal:      decimal.NewFromInt(100),
		PlatformFee:   decimal.NewFromInt(10),
		Tax:           decimal.NewFromInt(5),
		Total:         decimal.NewFromInt(115),
		Status:        domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[b1ID] = b1
	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(50), Status: domain.TicketStatusValid}
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(50), Status: domain.TicketStatusCancelled}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	f.txRepos.ticketRepo.tickets[t2.ID] = t2

	txID1 := "pay_cmplx_1"
	p1ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p1ID] = &domain.Payment{
		ID:            p1ID,
		BookingID:     b1ID.String(),
		Amount:        decimal.NewFromInt(115),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txID1,
	}

	// Prior customer refund of $50 (ticket 2 was cancelled earlier)
	r1ID := uuid.New()
	f.txRepos.refundRepo.refunds[r1ID] = &domain.Refund{
		ID:        r1ID,
		BookingID: b1ID,
		Amount:    decimal.NewFromInt(50),
		Status:    domain.RefundStatusSuccess,
		Reason:    "CUSTOMER_CANCELLATION",
	}

	// Booking 2: Free booking ($0)
	b2ID := uuid.New()
	b2 := &domain.Booking{
		ID:            b2ID,
		UserID:        uuid.New(),
		EventID:       f.eventID,
		BookingNumber: "BK-FREE",
		Total:         decimal.Zero,
		Status:        domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[b2ID] = b2

	// Booking 3: Failed payment ($80)
	b3ID := uuid.New()
	b3 := &domain.Booking{
		ID:            b3ID,
		UserID:        uuid.New(),
		EventID:       f.eventID,
		BookingNumber: "BK-FAIL",
		Total:         decimal.NewFromInt(80),
		Status:        domain.BookingStatusPending,
	}
	f.txRepos.bookingRepo.bookings[b3ID] = b3
	p3ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p3ID] = &domain.Payment{
		ID:        p3ID,
		BookingID: b3ID.String(),
		Amount:    decimal.NewFromInt(80),
		Status:    domain.PaymentStatusFailed,
	}

	preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
	require.NoError(t, err)

	// Total Bookings: 3; Affected Bookings: 2 (Confirmed)
	assert.Equal(t, 3, preview.OperationalImpact.TotalBookings)
	assert.Equal(t, 2, preview.OperationalImpact.AffectedBookings)
	assert.Equal(t, 1, preview.OperationalImpact.AffectedTickets)

	// Financials:
	// Gross Collected: Only successful payment p1 = $115 (p3 failed is excluded)
	assert.True(t, preview.FinancialImpact.GrossCollectedAmount.Equal(decimal.NewFromInt(115)))
	// Already refunded: r1 = $50
	assert.True(t, preview.FinancialImpact.AlreadyRefundedAmount.Equal(decimal.NewFromInt(50)))
	// Expected refund: Total $115 - AlreadyRefunded $50 = $65
	assert.True(t, preview.FinancialImpact.ExpectedRefundAmount.Equal(decimal.NewFromInt(65)))

	// Check ledger balance parity: platformFee + tax + organizerPayable == expectedRefund
	sumBreakdown := preview.FinancialImpact.PlatformFeeRefund.
		Add(preview.FinancialImpact.TaxRefund).
		Add(preview.FinancialImpact.OrganizerPayableRevocation)
	assert.True(t, sumBreakdown.Equal(preview.FinancialImpact.ExpectedRefundAmount))
}

func TestEventCancellationPreview_Authorization(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	t.Run("OtherOrganizerForbidden", func(t *testing.T) {
		otherUserID := uuid.New()
		otherOrgID := uuid.New()
		f.txRepos.organizerRepo.organizers[otherOrgID] = &domain.Organizer{
			ID:     otherOrgID,
			UserID: otherUserID,
			Status: "ACTIVE",
		}

		_, err := f.usecase.GetCancellationPreviewForOrganizer(ctx, otherUserID, f.eventID)
		require.Error(t, err)
		assert.ErrorIs(t, err, events.ErrUnauthorizedOrganizer)
	})

	t.Run("InactiveOrganizerForbidden", func(t *testing.T) {
		inactiveUserID := uuid.New()
		inactiveOrgID := uuid.New()
		f.txRepos.organizerRepo.organizers[inactiveOrgID] = &domain.Organizer{
			ID:     inactiveOrgID,
			UserID: inactiveUserID,
			Status: "SUSPENDED",
		}

		_, err := f.usecase.GetCancellationPreviewForOrganizer(ctx, inactiveUserID, f.eventID)
		require.Error(t, err)
		assert.ErrorIs(t, err, events.ErrUnauthorizedOrganizer)
	})

	t.Run("AdminCanAccessAnyEvent", func(t *testing.T) {
		preview, err := f.usecase.GetCancellationPreview(ctx, f.eventID)
		require.NoError(t, err)
		assert.Equal(t, f.eventID, preview.Event.ID)
	})

	t.Run("EventNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := f.usecase.GetCancellationPreview(ctx, nonExistentID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEventNotFound)
	})
}

func TestEventCancellationPreview_ConsistencyWithCancellation(t *testing.T) {
	ctx := context.Background()

	t.Run("ConsistentSuccessWhenEligible", func(t *testing.T) {
		f := newCancellationFixture(t)

		// 1. Check preview
		preview, err := f.usecase.GetCancellationPreviewForOrganizer(ctx, f.organizerUserID, f.eventID)
		require.NoError(t, err)
		require.True(t, preview.Eligibility.IsCancellable)

		// 2. Perform cancellation
		summary, cancelErr := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Test consistency")
		require.NoError(t, cancelErr)
		assert.Equal(t, domain.EventStatusCancelled, summary.EventStatus)
	})

	t.Run("ConsistentFailureWhenIneligible", func(t *testing.T) {
		f := newCancellationFixture(t)
		// Conclude event
		past := time.Now().Add(-24 * time.Hour)
		f.txRepos.scheduleRepo.schedules[f.eventID].EventDate = past
		f.txRepos.scheduleRepo.schedules[f.eventID].StartTime = past
		f.txRepos.scheduleRepo.schedules[f.eventID].EndTime = past.Add(2 * time.Hour)

		// 1. Check preview
		preview, err := f.usecase.GetCancellationPreviewForOrganizer(ctx, f.organizerUserID, f.eventID)
		require.NoError(t, err)
		assert.False(t, preview.Eligibility.IsCancellable)

		// 2. Cancellation must fail with matching domain error
		_, cancelErr := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Try cancel concluded")
		require.Error(t, cancelErr)
		assert.ErrorIs(t, cancelErr, domain.ErrPastEventCannotBeCancelled)
	})
}

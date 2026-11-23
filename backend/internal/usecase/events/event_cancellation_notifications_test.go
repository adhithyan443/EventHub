package events_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventCancellation_Notifications_CustomerAndOrganizerDelivered(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	custUserID := uuid.New()
	f.userRepo.users[custUserID] = &domain.User{
		ID:       custUserID,
		FullName: "Alice Customer",
		Email:    "alice@example.com",
		Role:     "CUSTOMER",
	}

	bookingID := uuid.New()
	refNum := "BK-ALICE-100"
	booking := &domain.Booking{
		ID:            bookingID,
		UserID:        custUserID,
		EventID:       f.eventID,
		BookingNumber: refNum,
		Status:        domain.BookingStatusConfirmed,
		Total:         decimal.NewFromInt(120),
		CreatedAt:     time.Now().Add(-2 * time.Hour),
	}
	f.txRepos.bookingRepo.bookings[bookingID] = booking

	ticket := &domain.Ticket{
		ID:        uuid.New(),
		BookingID: bookingID,
		EventID:   f.eventID,
		Status:    domain.TicketStatusValid,
		Price:     decimal.NewFromInt(120),
	}
	f.txRepos.ticketRepo.tickets[ticket.ID] = ticket

	gatewayTxID := "pay_test_alice"
	paymentID := uuid.New().String()
	payment := &domain.Payment{
		ID:            paymentID,
		BookingID:     bookingID.String(),
		Amount:        decimal.NewFromInt(120),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}
	f.txRepos.paymentRepo.payments[paymentID] = payment

	reason := "Severe storm damage to outdoor arena."
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	require.NoError(t, err)
	require.NotNil(t, summary)

	// Wait for background dispatch
	var custNotice domain.CustomerCancellationNotice
	var orgNotice domain.OrganizerCancellationNotice

	select {
	case custNotice = <-f.notifier.notifyCustCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for customer notification")
	}

	select {
	case orgNotice = <-f.notifier.notifyOrgCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for organizer notification")
	}

	// Verify Customer Notice
	assert.Equal(t, custUserID, custNotice.CustomerID)
	assert.Equal(t, "Alice Customer", custNotice.CustomerName)
	assert.Equal(t, "alice@example.com", custNotice.CustomerEmail)
	assert.Equal(t, f.eventID, custNotice.EventID)
	assert.Equal(t, f.event.Title, custNotice.EventTitle)
	assert.Equal(t, reason, custNotice.CancellationReason)
	assert.Equal(t, []string{refNum}, custNotice.BookingReferences)
	assert.Equal(t, 1, custNotice.TicketsCancelled)
	assert.True(t, custNotice.RefundAmount.Equal(decimal.NewFromInt(120)))
	assert.Equal(t, "PENDING", custNotice.RefundStatus)

	// Verify Organizer Notice
	assert.Equal(t, f.organizerID, orgNotice.OrganizerID)
	assert.Equal(t, "Test Organizer", orgNotice.OrganizerName)
	assert.Equal(t, "organizer@example.com", orgNotice.OrganizerEmail)
	assert.Equal(t, f.eventID, orgNotice.EventID)
	assert.Equal(t, f.event.Title, orgNotice.EventTitle)
	assert.Equal(t, reason, orgNotice.CancellationReason)
	assert.Equal(t, "organizer", orgNotice.ActorRole)
	assert.Equal(t, "Event Organizer", orgNotice.CancelledBy)
	assert.Equal(t, 1, orgNotice.TotalBookingsAffected)
	assert.Equal(t, 1, orgNotice.TotalTicketsAffected)
	assert.True(t, orgNotice.TotalRefundAmount.Equal(decimal.NewFromInt(120)))
	assert.Contains(t, orgNotice.SettlementImpact, "voided")
}

func TestEventCancellation_Notifications_ConsolidatedCustomerBookings(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	custUserID := uuid.New()
	f.userRepo.users[custUserID] = &domain.User{
		ID:       custUserID,
		FullName: "Multi Booking User",
		Email:    "multi@example.com",
		Role:     "CUSTOMER",
	}

	// Booking 1: 2 tickets, $100
	b1ID := uuid.New()
	b1 := &domain.Booking{
		ID:            b1ID,
		UserID:        custUserID,
		EventID:       f.eventID,
		BookingNumber: "BK-MULTI-1",
		Status:        domain.BookingStatusConfirmed,
		Total:         decimal.NewFromInt(100),
	}
	f.txRepos.bookingRepo.bookings[b1ID] = b1
	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	f.txRepos.ticketRepo.tickets[t2.ID] = t2
	tx1 := "pay_test_multi_1"
	p1ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p1ID] = &domain.Payment{ID: p1ID, BookingID: b1ID.String(), Amount: decimal.NewFromInt(100), Status: domain.PaymentStatusSuccess, TransactionID: &tx1}

	// Booking 2: 1 ticket, $50
	b2ID := uuid.New()
	b2 := &domain.Booking{
		ID:            b2ID,
		UserID:        custUserID,
		EventID:       f.eventID,
		BookingNumber: "BK-MULTI-2",
		Status:        domain.BookingStatusConfirmed,
		Total:         decimal.NewFromInt(50),
	}
	f.txRepos.bookingRepo.bookings[b2ID] = b2
	t3 := &domain.Ticket{ID: uuid.New(), BookingID: b2ID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	f.txRepos.ticketRepo.tickets[t3.ID] = t3
	tx2 := "pay_test_multi_2"
	p2ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p2ID] = &domain.Payment{ID: p2ID, BookingID: b2ID.String(), Amount: decimal.NewFromInt(50), Status: domain.PaymentStatusSuccess, TransactionID: &tx2}

	reason := "Emergency facility maintenance required."
	_, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	require.NoError(t, err)

	var custNotice domain.CustomerCancellationNotice
	select {
	case custNotice = <-f.notifier.notifyCustCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for consolidated customer notice")
	}

	// Ensure no extra customer notices are sent
	select {
	case extra := <-f.notifier.notifyCustCh:
		t.Fatalf("unexpected duplicate customer notice received: %+v", extra)
	case <-time.After(100 * time.Millisecond):
	}

	assert.Equal(t, "multi@example.com", custNotice.CustomerEmail)
	assert.Equal(t, 2, len(custNotice.BookingReferences))
	assert.Contains(t, custNotice.BookingReferences, "BK-MULTI-1")
	assert.Contains(t, custNotice.BookingReferences, "BK-MULTI-2")
	assert.Equal(t, 3, custNotice.TicketsCancelled)
	assert.True(t, custNotice.RefundAmount.Equal(decimal.NewFromInt(150)))
}

func TestEventCancellation_Notifications_MultipleCustomers(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	cust1 := uuid.New()
	f.userRepo.users[cust1] = &domain.User{ID: cust1, FullName: "Customer One", Email: "c1@example.com"}
	b1ID := uuid.New()
	f.txRepos.bookingRepo.bookings[b1ID] = &domain.Booking{ID: b1ID, UserID: cust1, EventID: f.eventID, BookingNumber: "BK-C1", Status: domain.BookingStatusConfirmed, Total: decimal.NewFromInt(50)}
	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	txC1 := "pay_test_c1"
	p1ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p1ID] = &domain.Payment{ID: p1ID, BookingID: b1ID.String(), Amount: decimal.NewFromInt(50), Status: domain.PaymentStatusSuccess, TransactionID: &txC1}

	cust2 := uuid.New()
	f.userRepo.users[cust2] = &domain.User{ID: cust2, FullName: "Customer Two", Email: "c2@example.com"}
	b2ID := uuid.New()
	f.txRepos.bookingRepo.bookings[b2ID] = &domain.Booking{ID: b2ID, UserID: cust2, EventID: f.eventID, BookingNumber: "BK-C2", Status: domain.BookingStatusConfirmed, Total: decimal.NewFromInt(80)}
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: b2ID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(80)}
	f.txRepos.ticketRepo.tickets[t2.ID] = t2
	txC2 := "pay_test_c2"
	p2ID := uuid.New().String()
	f.txRepos.paymentRepo.payments[p2ID] = &domain.Payment{ID: p2ID, BookingID: b2ID.String(), Amount: decimal.NewFromInt(80), Status: domain.PaymentStatusSuccess, TransactionID: &txC2}

	reason := "Tour cancellation by producer."
	_, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	require.NoError(t, err)

	receivedEmails := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case notice := <-f.notifier.notifyCustCh:
			receivedEmails[notice.CustomerEmail] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for multiple customer notices")
		}
	}

	assert.True(t, receivedEmails["c1@example.com"])
	assert.True(t, receivedEmails["c2@example.com"])

	select {
	case orgNotice := <-f.notifier.notifyOrgCh:
		assert.Equal(t, 2, orgNotice.TotalBookingsAffected)
		assert.Equal(t, 2, orgNotice.TotalTicketsAffected)
		assert.True(t, orgNotice.TotalRefundAmount.Equal(decimal.NewFromInt(130)))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for organizer notice")
	}
}

func TestEventCancellation_Notifications_CustomerFailureDoesNotBlockOrganizer(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Make customer notifications return an error
	f.notifier.customerErr = errors.New("smtp connection refused")

	custID := uuid.New()
	f.userRepo.users[custID] = &domain.User{ID: custID, FullName: "User", Email: "u@example.com"}
	bID := uuid.New()
	f.txRepos.bookingRepo.bookings[bID] = &domain.Booking{ID: bID, UserID: custID, EventID: f.eventID, BookingNumber: "BK-ERR", Status: domain.BookingStatusConfirmed, Total: decimal.NewFromInt(50)}
	tID := uuid.New()
	f.txRepos.ticketRepo.tickets[tID] = &domain.Ticket{ID: tID, BookingID: bID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	pID := uuid.New().String()
	f.txRepos.paymentRepo.payments[pID] = &domain.Payment{ID: pID, BookingID: bID.String(), Amount: decimal.NewFromInt(50), Status: domain.PaymentStatusSuccess}

	reason := "Cancelled due to scheduling conflict."
	_, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	// Cancellation MUST still succeed
	require.NoError(t, err)

	// Customer notification attempt should happen
	select {
	case <-f.notifier.notifyCustCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for customer notification attempt")
	}

	// Organizer notification MUST still be attempted and delivered!
	select {
	case orgNotice := <-f.notifier.notifyOrgCh:
		assert.Equal(t, "organizer@example.com", orgNotice.OrganizerEmail)
	case <-time.After(2 * time.Second):
		t.Fatal("organizer notification was blocked by customer notification failure")
	}
}

func TestEventCancellation_Notifications_OrganizerFailureDoesNotBlockCustomers(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Make organizer notification return an error
	f.notifier.organizerErr = errors.New("smtp timeout")

	custID := uuid.New()
	f.userRepo.users[custID] = &domain.User{ID: custID, FullName: "User", Email: "u@example.com"}
	bID := uuid.New()
	f.txRepos.bookingRepo.bookings[bID] = &domain.Booking{ID: bID, UserID: custID, EventID: f.eventID, BookingNumber: "BK-OK", Status: domain.BookingStatusConfirmed, Total: decimal.NewFromInt(50)}
	tID := uuid.New()
	f.txRepos.ticketRepo.tickets[tID] = &domain.Ticket{ID: tID, BookingID: bID, EventID: f.eventID, Status: domain.TicketStatusValid, Price: decimal.NewFromInt(50)}
	pID := uuid.New().String()
	f.txRepos.paymentRepo.payments[pID] = &domain.Payment{ID: pID, BookingID: bID.String(), Amount: decimal.NewFromInt(50), Status: domain.PaymentStatusSuccess}

	reason := "Cancelled due to technical difficulties."
	_, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	// Cancellation MUST still succeed
	require.NoError(t, err)

	// Customer notice delivered
	select {
	case custNotice := <-f.notifier.notifyCustCh:
		assert.Equal(t, "u@example.com", custNotice.CustomerEmail)
	case <-time.After(2 * time.Second):
		t.Fatal("customer notification was blocked")
	}

	// Organizer notice attempted
	select {
	case <-f.notifier.notifyOrgCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for organizer notice attempt")
	}
}

func TestEventCancellation_Notifications_AdminCancellationNotice(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	adminUserID := uuid.New()
	reason := "Terms of service violation."

	summary, err := f.usecase.CancelEventByAdmin(ctx, adminUserID, f.eventID, reason)
	require.NoError(t, err)
	require.NotNil(t, summary)

	select {
	case orgNotice := <-f.notifier.notifyOrgCh:
		assert.Equal(t, "admin", orgNotice.ActorRole)
		assert.Equal(t, "Platform Administrator", orgNotice.CancelledBy)
		assert.Equal(t, reason, orgNotice.CancellationReason)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for admin cancellation organizer notice")
	}
}

func TestEventCancellation_Notifications_NoDispatchOnTransactionFailure(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Inject settlement repository error
	f.txRepos.settlementRepo.updateErr = errors.New("simulated database failure during settlement rejection")

	// Add pending settlement that triggers update
	settlementID := uuid.New()
	pendingSettlement := &domain.Settlement{
		ID:          settlementID,
		OrganizerID: f.organizerID,
		EventID:     &f.eventID,
		Amount:      decimal.NewFromInt(1500),
		Currency:    "INR",
		Status:      domain.SettlementStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	f.txRepos.settlementRepo.settlements[settlementID] = pendingSettlement

	reason := "Cancelled"
	_, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	require.Error(t, err)

	// Verify no notifications were dispatched
	select {
	case n := <-f.notifier.notifyCustCh:
		t.Fatalf("customer notice should not have been dispatched on rollback: %+v", n)
	case n := <-f.notifier.notifyOrgCh:
		t.Fatalf("organizer notice should not have been dispatched on rollback: %+v", n)
	case <-time.After(150 * time.Millisecond):
		// Success: nothing dispatched
	}
}

func TestEventCancellation_Notifications_ShutdownRejectionSafety(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Create a tracker and close it immediately (simulating shutdown)
	tracker := worker.NewDispatchTracker()
	tracker.Close()
	f.usecase.SetDispatchTracker(tracker)

	reason := "Post-concluded cancellation attempt"
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	// Cancellation itself should still succeed gracefully even when background notifications are rejected by stopped tracker
	require.NoError(t, err)
	require.NotNil(t, summary)

	// Since tracker is stopped, no background notifications are dispatched
	select {
	case n := <-f.notifier.notifyCustCh:
		t.Fatalf("unexpected notice dispatched: %+v", n)
	case n := <-f.notifier.notifyOrgCh:
		t.Fatalf("unexpected notice dispatched: %+v", n)
	case <-time.After(150 * time.Millisecond):
	}
}

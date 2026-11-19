package events_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Mocks ---

type mockTxManager struct {
	repos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

type mockTxRepos struct {
	domain.TransactionRepositories
	eventRepo            *mockEventRepo
	cancellationRepo     *mockCancellationRepo
	organizerRepo        *mockOrganizerRepo
	bookingRepo          *mockBookingRepo
	ticketRepo           *mockTicketRepo
	paymentRepo          *mockPaymentRepo
	refundRepo           *mockRefundRepo
}

func (m *mockTxRepos) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockTxRepos) BookingRepository() domain.BookingRepository     { return m.bookingRepo }
func (m *mockTxRepos) TicketRepository() domain.TicketRepository       { return m.ticketRepo }
func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository     { return m.paymentRepo }
func (m *mockTxRepos) RefundRepository() domain.RefundRepository       { return m.refundRepo }

type mockEventRepo struct {
	domain.EventRepository
	mu                 sync.Mutex
	events             map[uuid.UUID]*domain.Event
	findForUpdateCalls int
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{events: make(map[uuid.UUID]*domain.Event)}
}

func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ev, ok := m.events[id]; ok {
		copied := *ev
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}

func (m *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	m.mu.Lock()
	m.findForUpdateCalls++
	m.mu.Unlock()
	return m.FindByID(id)
}

func (m *mockEventRepo) Update(event *domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.events[event.ID]; !ok {
		return domain.ErrEventNotFound
	}
	copied := *event
	m.events[event.ID] = &copied
	return nil
}

type mockCancellationRepo struct {
	domain.EventCancellationRepository
	mu            sync.Mutex
	cancellations map[uuid.UUID]*domain.EventCancellation
}

func newMockCancellationRepo() *mockCancellationRepo {
	return &mockCancellationRepo{cancellations: make(map[uuid.UUID]*domain.EventCancellation)}
}

func (m *mockCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.cancellations[eventID]; ok {
		copied := *c
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}

func (m *mockCancellationRepo) Create(c *domain.EventCancellation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *c
	m.cancellations[c.EventID] = &copied
	return nil
}

func (m *mockCancellationRepo) Update(c *domain.EventCancellation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *c
	m.cancellations[c.EventID] = &copied
	return nil
}

type mockOrganizerRepo struct {
	domain.OrganizerRepository
	mu         sync.Mutex
	organizers map[uuid.UUID]*domain.Organizer
}

func newMockOrganizerRepo() *mockOrganizerRepo {
	return &mockOrganizerRepo{organizers: make(map[uuid.UUID]*domain.Organizer)}
}

func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, org := range m.organizers {
		if org.UserID == userID {
			copied := *org
			return &copied, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockBookingRepo struct {
	domain.BookingRepository
	mu       sync.Mutex
	bookings map[uuid.UUID]*domain.Booking
}

func newMockBookingRepo() *mockBookingRepo {
	return &mockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)}
}

func (m *mockBookingRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Booking
	for _, b := range m.bookings {
		if b.EventID == eventID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[id]
	if !ok {
		return domain.ErrBookingNotFound
	}
	b.Status = status
	return nil
}

type mockTicketRepo struct {
	domain.TicketRepository
	mu      sync.Mutex
	tickets map[uuid.UUID]*domain.Ticket
}

func newMockTicketRepo() *mockTicketRepo {
	return &mockTicketRepo{tickets: make(map[uuid.UUID]*domain.Ticket)}
}

func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Ticket
	for _, t := range m.tickets {
		if t.BookingID == bookingID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTicketRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Ticket
	for _, t := range m.tickets {
		if t.EventID == eventID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTicketRepo) UpdateStatusForTicketIDs(ctx context.Context, ticketIDs []uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ticketIDs {
		t, ok := m.tickets[id]
		if !ok || (status == domain.TicketStatusCancelled && t.Status != domain.TicketStatusValid) {
			return domain.ErrTicketNotValid
		}
		t.Status = status
	}
	return nil
}

type mockPaymentRepo struct {
	domain.PaymentRepository
	mu       sync.Mutex
	payments map[string]*domain.Payment
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
}

func (m *mockPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Payment
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			result = append(result, *p)
		}
	}
	return result, nil
}

type mockRefundRepo struct {
	domain.RefundRepository
	mu      sync.Mutex
	refunds map[uuid.UUID]*domain.Refund
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{refunds: make(map[uuid.UUID]*domain.Refund)}
}

func (m *mockRefundRepo) Create(ctx context.Context, r *domain.Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *r
	m.refunds[r.ID] = &copied
	return nil
}

func (m *mockRefundRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Refund
	for _, r := range m.refunds {
		result = append(result, *r)
	}
	return result, nil
}

type mockRefundProcessor struct {
	mu           sync.Mutex
	processedIDs []uuid.UUID
	processErr   error
}

func (m *mockRefundProcessor) ProcessPendingRefund(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedIDs = append(m.processedIDs, refundID)
	return nil, m.processErr
}

func (m *mockRefundProcessor) ReconcileStuckRefunds(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	return 0, nil
}
func (m *mockRefundProcessor) GetRefundByID(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error) {
	return nil, nil
}
func (m *mockRefundProcessor) GetRefundsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	return nil, nil
}

// --- Test Fixture ---

type cancellationFixture struct {
	txManager       *mockTxManager
	txRepos         *mockTxRepos
	refundProcessor *mockRefundProcessor
	usecase         *events.EventCancellationUsecase
	organizerUserID uuid.UUID
	organizerID     uuid.UUID
	eventID         uuid.UUID
	event           *domain.Event
}

func newCancellationFixture(t *testing.T) *cancellationFixture {
	t.Helper()

	orgUserID := uuid.New()
	orgID := uuid.New()
	evID := uuid.New()

	eventRepo := newMockEventRepo()
	cancellationRepo := newMockCancellationRepo()
	organizerRepo := newMockOrganizerRepo()
	bookingRepo := newMockBookingRepo()
	ticketRepo := newMockTicketRepo()
	paymentRepo := newMockPaymentRepo()
	refundRepo := newMockRefundRepo()

	txRepos := &mockTxRepos{
		eventRepo:        eventRepo,
		cancellationRepo: cancellationRepo,
		organizerRepo:    organizerRepo,
		bookingRepo:      bookingRepo,
		ticketRepo:       ticketRepo,
		paymentRepo:      paymentRepo,
		refundRepo:       refundRepo,
	}

	txMgr := &mockTxManager{repos: txRepos}
	proc := &mockRefundProcessor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := events.NewEventCancellationUsecase(txMgr, proc, logger)

	// Setup active organizer
	organizer := &domain.Organizer{
		ID:     orgID,
		UserID: orgUserID,
		Status: "ACTIVE",
	}
	organizerRepo.organizers[orgID] = organizer

	// Setup published event
	now := time.Now()
	event := &domain.Event{
		ID:          evID,
		OrganizerID: orgID,
		Title:       "Tech World Summit 2026",
		Status:      domain.EventStatusPublished,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now.Add(-24 * time.Hour),
	}
	eventRepo.events[evID] = event

	cancellationSetting := &domain.EventCancellation{
		ID:                  uuid.New(),
		EventID:             evID,
		CancellationAllowed: true,
		RefundPolicy:        domain.RefundPolicyFull,
		RefundPercentage:    100,
	}
	cancellationRepo.cancellations[evID] = cancellationSetting

	return &cancellationFixture{
		txManager:       txMgr,
		txRepos:         txRepos,
		refundProcessor: proc,
		usecase:         uc,
		organizerUserID: orgUserID,
		organizerID:     orgID,
		eventID:         evID,
		event:           event,
	}
}

// --- Unit Tests ---

func TestEventCancellationUsecase_CancelEventByOrganizer_Success_FullRefunds(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Booking 1: Subtotal 200, Fee 10, Tax 2, Total 212 (2 tickets)
	b1ID := uuid.New()
	p1ID := uuid.New()
	gatewayTxID := "pay_test_event_1"

	b1 := &domain.Booking{
		ID:          b1ID,
		EventID:     f.eventID,
		Subtotal:    decimal.NewFromInt(200),
		PlatformFee: decimal.NewFromInt(10),
		Tax:         decimal.NewFromInt(2),
		Total:       decimal.NewFromInt(212),
		Status:      domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[b1ID] = b1

	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(100), Status: domain.TicketStatusValid}
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(100), Status: domain.TicketStatusValid}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	f.txRepos.ticketRepo.tickets[t2.ID] = t2

	p1 := &domain.Payment{
		ID:            p1ID.String(),
		BookingID:     b1ID.String(),
		Amount:        decimal.NewFromInt(212),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}
	f.txRepos.paymentRepo.payments[p1ID.String()] = p1

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Venue damaged by flood")
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, f.eventID, summary.EventID)
	assert.Equal(t, domain.EventStatusCancelled, summary.EventStatus)
	assert.Equal(t, "Venue damaged by flood", summary.CancellationReason)
	assert.Equal(t, 1, summary.TotalBookingsAffected)
	assert.Equal(t, 2, summary.TotalTicketsAffected)
	// Full refund including ticket price + platform fee + tax
	assert.True(t, summary.TotalEligibleRefundAmount.Equal(decimal.NewFromInt(212)))
	assert.Equal(t, 1, summary.PendingRefundCount)

	// Verify Event status in DB
	ev, err := f.txRepos.eventRepo.FindByID(f.eventID)
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusCancelled, ev.Status)

	// Verify Tickets status in DB
	tk1 := f.txRepos.ticketRepo.tickets[t1.ID]
	tk2 := f.txRepos.ticketRepo.tickets[t2.ID]
	assert.Equal(t, domain.TicketStatusCancelled, tk1.Status)
	assert.Equal(t, domain.TicketStatusCancelled, tk2.Status)

	// Verify Booking status in DB
	bk := f.txRepos.bookingRepo.bookings[b1ID]
	assert.Equal(t, domain.BookingStatusCancelled, bk.Status)

	// Verify Refund record created in PENDING
	require.Len(t, f.txRepos.refundRepo.refunds, 1)
	var createdRefund *domain.Refund
	for _, r := range f.txRepos.refundRepo.refunds {
		createdRefund = r
		break
	}
	assert.Equal(t, domain.RefundStatusPending, createdRefund.Status)
	assert.Equal(t, "EVENT_CANCELLATION", createdRefund.Reason)
	assert.True(t, createdRefund.Amount.Equal(decimal.NewFromInt(212)))

	// Allow goroutine execution and verify async processor was called
	time.Sleep(50 * time.Millisecond)
	f.refundProcessor.mu.Lock()
	assert.Contains(t, f.refundProcessor.processedIDs, createdRefund.ID)
	f.refundProcessor.mu.Unlock()
}

func TestEventCancellationUsecase_CancelEventByAdmin_Success(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()
	adminID := uuid.New()

	summary, err := f.usecase.CancelEventByAdmin(ctx, adminID, f.eventID, "Safety inspection failed")
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, domain.EventStatusCancelled, summary.EventStatus)
	assert.Equal(t, "Safety inspection failed", summary.CancellationReason)

	ev, err := f.txRepos.eventRepo.FindByID(f.eventID)
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusCancelled, ev.Status)
}

func TestEventCancellationUsecase_CancelEventByOrganizer_Unauthorized_DifferentOrganizer(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	otherOrgUserID := uuid.New()
	otherOrgID := uuid.New()
	f.txRepos.organizerRepo.organizers[otherOrgID] = &domain.Organizer{
		ID:     otherOrgID,
		UserID: otherOrgUserID,
		Status: "ACTIVE",
	}

	summary, err := f.usecase.CancelEventByOrganizer(ctx, otherOrgUserID, f.eventID, "I want to cancel this")
	require.Error(t, err)
	assert.True(t, errors.Is(err, events.ErrUnauthorizedOrganizer))
	assert.Nil(t, summary)

	// Verify Event was NOT cancelled
	ev, err := f.txRepos.eventRepo.FindByID(f.eventID)
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusPublished, ev.Status)
}

func TestEventCancellationUsecase_CancelEventByOrganizer_InactiveOrganizer_Rejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Set organizer inactive
	f.txRepos.organizerRepo.organizers[f.organizerID].Status = "SUSPENDED"

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Cancel")
	require.Error(t, err)
	assert.True(t, errors.Is(err, events.ErrUnauthorizedOrganizer))
	assert.Nil(t, summary)
}

func TestEventCancellationUsecase_CancelEvent_Idempotency(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// 1st cancel call
	sum1, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Initial cancellation")
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusCancelled, sum1.EventStatus)

	// 2nd cancel call (idempotent)
	sum2, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Repeated cancel call")
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusCancelled, sum2.EventStatus)
	assert.Equal(t, sum1.CancelledAt, sum2.CancelledAt)
	assert.Equal(t, 0, len(f.txRepos.refundRepo.refunds)) // No tickets existed, 0 refunds created
}

func TestEventCancellationUsecase_CancelEvent_PartialCustomerCancellation_RemainingTicketsRefunded(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Booking originally had 2 tickets. Total 112 (100 subtotal + 10 fee + 2 tax).
	// Customer already cancelled Ticket 1 earlier, receiving 50 refund (without fee).
	bID := uuid.New()
	pID := uuid.New()
	gatewayTxID := "pay_test_event_partial"

	b := &domain.Booking{
		ID:          bID,
		EventID:     f.eventID,
		Subtotal:    decimal.NewFromInt(100),
		PlatformFee: decimal.NewFromInt(10),
		Tax:         decimal.NewFromInt(2),
		Total:       decimal.NewFromInt(112),
		Status:      domain.BookingStatusPartiallyCancelled,
	}
	f.txRepos.bookingRepo.bookings[bID] = b

	// Ticket 1: Already cancelled
	t1 := &domain.Ticket{ID: uuid.New(), BookingID: bID, EventID: f.eventID, Price: decimal.NewFromInt(50), Status: domain.TicketStatusCancelled}
	// Ticket 2: Still valid
	t2 := &domain.Ticket{ID: uuid.New(), BookingID: bID, EventID: f.eventID, Price: decimal.NewFromInt(50), Status: domain.TicketStatusValid}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1
	f.txRepos.ticketRepo.tickets[t2.ID] = t2

	// Pre-existing customer cancellation refund of 50
	rCustomer := &domain.Refund{
		ID:        uuid.New(),
		BookingID: bID,
		PaymentID: pID,
		Amount:    decimal.NewFromInt(50),
		Status:    domain.RefundStatusSuccess,
		Reason:    "CUSTOMER_CANCELLATION",
	}
	f.txRepos.refundRepo.refunds[rCustomer.ID] = rCustomer

	p := &domain.Payment{
		ID:            pID.String(),
		BookingID:     bID.String(),
		Amount:        decimal.NewFromInt(112),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}
	f.txRepos.paymentRepo.payments[pID.String()] = p

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Cancelled")
	require.NoError(t, err)

	// Remaining refund should be 112 - 50 = 62 (remaining ticket price 50 + platform fee 10 + tax 2)
	assert.True(t, summary.TotalEligibleRefundAmount.Equal(decimal.NewFromInt(62)))

	// Check the new refund created for EVENT_CANCELLATION
	var eventRefund *domain.Refund
	for _, r := range f.txRepos.refundRepo.refunds {
		if r.Reason == "EVENT_CANCELLATION" {
			eventRefund = r
			break
		}
	}
	require.NotNil(t, eventRefund)
	assert.True(t, eventRefund.Amount.Equal(decimal.NewFromInt(62)))
	assert.Equal(t, domain.RefundStatusPending, eventRefund.Status)

	// Verify both tickets are now cancelled
	assert.Equal(t, domain.TicketStatusCancelled, f.txRepos.ticketRepo.tickets[t2.ID].Status)
	assert.Equal(t, domain.BookingStatusCancelled, f.txRepos.bookingRepo.bookings[bID].Status)
}

func TestEventCancellationUsecase_CancelEvent_FreeBooking_NoRefundCreated(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	bID := uuid.New()
	b := &domain.Booking{
		ID:       bID,
		EventID:  f.eventID,
		Subtotal: decimal.Zero,
		Total:    decimal.Zero,
		Status:   domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[bID] = b

	t1 := &domain.Ticket{ID: uuid.New(), BookingID: bID, EventID: f.eventID, Price: decimal.Zero, Status: domain.TicketStatusValid}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Free event cancelled")
	require.NoError(t, err)

	assert.Equal(t, domain.EventStatusCancelled, summary.EventStatus)
	assert.True(t, summary.TotalEligibleRefundAmount.IsZero())
	assert.Equal(t, 0, summary.PendingRefundCount)
	assert.Equal(t, "NO_REFUNDS_NEEDED", summary.CompletionStatus)

	// Ticket and booking should be marked CANCELLED
	assert.Equal(t, domain.TicketStatusCancelled, f.txRepos.ticketRepo.tickets[t1.ID].Status)
	assert.Equal(t, domain.BookingStatusCancelled, f.txRepos.bookingRepo.bookings[bID].Status)
	assert.Empty(t, f.txRepos.refundRepo.refunds)
}

func TestEventCancellationUsecase_GetEventCancellationStatus_TrackingMetrics(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Pre-populate refund states
	r1 := &domain.Refund{ID: uuid.New(), Amount: decimal.NewFromInt(100), Status: domain.RefundStatusSuccess, Reason: "EVENT_CANCELLATION"}
	r2 := &domain.Refund{ID: uuid.New(), Amount: decimal.NewFromInt(50), Status: domain.RefundStatusProcessing, Reason: "EVENT_CANCELLATION"}
	r3 := &domain.Refund{ID: uuid.New(), Amount: decimal.NewFromInt(30), Status: domain.RefundStatusFailed, Reason: "EVENT_CANCELLATION"}
	f.txRepos.refundRepo.refunds[r1.ID] = r1
	f.txRepos.refundRepo.refunds[r2.ID] = r2
	f.txRepos.refundRepo.refunds[r3.ID] = r3

	f.event.Status = domain.EventStatusCancelled
	f.txRepos.eventRepo.events[f.eventID] = f.event

	summary, err := f.usecase.GetEventCancellationStatus(ctx, f.eventID)
	require.NoError(t, err)

	assert.True(t, summary.TotalEligibleRefundAmount.Equal(decimal.NewFromInt(180)))
	assert.True(t, summary.TotalRefundedAmount.Equal(decimal.NewFromInt(100)))
	assert.True(t, summary.PendingRefundAmount.Equal(decimal.NewFromInt(50)))
	assert.True(t, summary.FailedRefundAmount.Equal(decimal.NewFromInt(30)))
	assert.Equal(t, 1, summary.PendingRefundCount)
	assert.Equal(t, 1, summary.FailedRefundCount)
	assert.Equal(t, 1, summary.SuccessfulRefundCount)
	assert.Equal(t, "PROCESSING", summary.CompletionStatus)
}

func TestEventCancellationUsecase_GetCancellationStatusForOrganizer_Ownership(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Owner can access
	s, err := f.usecase.GetEventCancellationStatusForOrganizer(ctx, f.organizerUserID, f.eventID)
	require.NoError(t, err)
	assert.NotNil(t, s)

	// Different organizer is rejected
	otherUserID := uuid.New()
	otherOrgID := uuid.New()
	f.txRepos.organizerRepo.organizers[otherOrgID] = &domain.Organizer{ID: otherOrgID, UserID: otherUserID, Status: "ACTIVE"}

	s, err = f.usecase.GetEventCancellationStatusForOrganizer(ctx, otherUserID, f.eventID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, events.ErrUnauthorizedOrganizer))
	assert.Nil(t, s)
}

func TestEventCancellationUsecase_SingleEventRowLockQuery_NoRedundantQuery(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Perform cancellation by organizer
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Test single row lock")
	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Verify FindByIDForUpdate was called exactly once, avoiding duplicate locking queries in the transaction
	f.txRepos.eventRepo.mu.Lock()
	defer f.txRepos.eventRepo.mu.Unlock()
	assert.Equal(t, 1, f.txRepos.eventRepo.findForUpdateCalls, "FindByIDForUpdate must be called exactly once during organizer cancellation")
}

func TestEventCancellationUsecase_GracefulShutdown_TracksBackgroundRefundDispatch(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Add a booking with valid ticket and payment
	b1ID := uuid.New()
	p1ID := uuid.New()
	gatewayTxID := "pay_test_event_shutdown"

	b1 := &domain.Booking{
		ID:          b1ID,
		EventID:     f.eventID,
		Subtotal:    decimal.NewFromInt(200),
		PlatformFee: decimal.NewFromInt(10),
		Tax:         decimal.NewFromInt(2),
		Total:       decimal.NewFromInt(212),
		Status:      domain.BookingStatusConfirmed,
	}
	f.txRepos.bookingRepo.bookings[b1ID] = b1

	t1 := &domain.Ticket{ID: uuid.New(), BookingID: b1ID, EventID: f.eventID, Price: decimal.NewFromInt(100), Status: domain.TicketStatusValid}
	f.txRepos.ticketRepo.tickets[t1.ID] = t1

	p1 := &domain.Payment{
		ID:            p1ID.String(),
		BookingID:     b1ID.String(),
		Amount:        decimal.NewFromInt(212),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}
	f.txRepos.paymentRepo.payments[p1ID.String()] = p1

	tracker := worker.NewDispatchTracker()
	f.usecase.SetDispatchTracker(tracker)

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Shutdown test")
	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Verify that dispatchTracker tracks the background refund dispatch and completes
	done := make(chan struct{})
	go func() {
		tracker.CloseAndWait()
		close(done)
	}()

	select {
	case <-done:
		// Success: tracked properly
	case <-time.After(2 * time.Second):
		t.Fatal("dispatchTracker timed out waiting for background refund processing")
	}

	f.refundProcessor.mu.Lock()
	defer f.refundProcessor.mu.Unlock()
	assert.NotEmpty(t, f.refundProcessor.processedIDs, "Background refund processor should have processed refunds")
}

func TestEventCancellationUsecase_ShutdownActive_DispatchRejected_RefundRemainsPending(t *testing.T) {
	ctx := context.Background()
	f := newCancellationFixture(t)

	b1ID := uuid.New()
	p1ID := uuid.New()
	f.txRepos.bookingRepo.bookings[b1ID] = &domain.Booking{
		ID:          b1ID,
		EventID:     f.eventID,
		UserID:      uuid.New(),
		Status:      domain.BookingStatusConfirmed,
		Total:       decimal.NewFromInt(212),
		PlatformFee: decimal.NewFromInt(10),
		Tax:         decimal.NewFromInt(2),
	}
	t1ID := uuid.New()
	f.txRepos.ticketRepo.tickets[t1ID] = &domain.Ticket{
		ID:        t1ID,
		BookingID: b1ID,
		EventID:   f.eventID,
		Price:     decimal.NewFromInt(100),
		Status:    domain.TicketStatusValid,
	}

	gatewayTxID := "pay_shutdown_closed_111"
	p1 := &domain.Payment{
		ID:            p1ID.String(),
		BookingID:     b1ID.String(),
		Amount:        decimal.NewFromInt(212),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}
	f.txRepos.paymentRepo.payments[p1ID.String()] = p1

	tracker := worker.NewDispatchTracker()
	// Simulate shutdown already initiated
	tracker.Close()
	f.usecase.SetDispatchTracker(tracker)

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Shutdown already closed")
	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Since tracker was closed, immediate background dispatch must have been rejected
	f.refundProcessor.mu.Lock()
	assert.Empty(t, f.refundProcessor.processedIDs, "Immediate dispatch must be skipped when shutdown is active")
	f.refundProcessor.mu.Unlock()

	// But the refund record in DB must remain durable in PENDING status for reconciliation recovery
	f.txRepos.refundRepo.mu.Lock()
	defer f.txRepos.refundRepo.mu.Unlock()
	require.NotEmpty(t, f.txRepos.refundRepo.refunds, "Refund record must be created in DB")

	var pendingRefundFound bool
	for _, ref := range f.txRepos.refundRepo.refunds {
		if ref.BookingID == b1ID && ref.Status == domain.RefundStatusPending && ref.Reason == "EVENT_CANCELLATION" {
			pendingRefundFound = true
			break
		}
	}
	assert.True(t, pendingRefundFound, "Refund must remain in PENDING status in DB awaiting reconciliation worker")
}

package payment_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	bookinguc "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations ---

type mockPaymentRepo struct {
	domain.PaymentRepository
	mu           sync.Mutex
	payments     map[string]*domain.Payment
	createCalls  int
	failCalls    int
	refundCalls  int
	successCalls int
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		payments: make(map[string]*domain.Payment),
	}
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls++
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	copied := *p
	m.payments[p.ID] = &copied
	return nil
}

func (m *mockPaymentRepo) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.payments[id]; ok {
		copied := *p
		return &copied, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByIDForUpdate(ctx context.Context, id string) (*domain.Payment, error) {
	return m.FindByID(ctx, id)
}

func (m *mockPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Payment
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockPaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.payments[p.ID]; ok {
		*existing = *p
		return nil
	}
	return domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.payments {
		if p.ProviderOrderID != nil && *p.ProviderOrderID == providerOrderID {
			copied := *p
			return &copied, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByProviderOrderIDForUpdate(ctx context.Context, providerOrderID string) (*domain.Payment, error) {
	return m.FindByProviderOrderID(ctx, providerOrderID)
}

func (m *mockPaymentRepo) MarkSuccessful(ctx context.Context, paymentID string, transactionID string, paidAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successCalls++
	if p, ok := m.payments[paymentID]; ok {
		if p.Status != domain.PaymentStatusPending {
			return domain.ErrPaymentAlreadyComplete
		}
		p.Status = domain.PaymentStatusSuccess
		p.TransactionID = &transactionID
		p.PaidAt = &paidAt
		return nil
	}
	return domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) MarkFailed(ctx context.Context, paymentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCalls++
	if p, ok := m.payments[paymentID]; ok {
		if p.Status == domain.PaymentStatusPending {
			p.Status = domain.PaymentStatusFailed
		}
		return nil
	}
	return domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) MarkRefundRequired(ctx context.Context, paymentID string, transactionID string, paidAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refundCalls++
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if p.Status == domain.PaymentStatusRefundRequired {
		if p.TransactionID != nil && *p.TransactionID == transactionID {
			return nil
		}
		return domain.ErrInvalidPayment
	}
	if p.Status == domain.PaymentStatusSuccess {
		return domain.ErrPaymentAlreadyComplete
	}
	if p.Status != domain.PaymentStatusPending {
		return domain.ErrInvalidPayment
	}
	p.Status = domain.PaymentStatusRefundRequired
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

func (m *mockPaymentRepo) UpdateProviderOrderID(ctx context.Context, paymentID string, providerOrderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if p.ProviderOrderID != nil && *p.ProviderOrderID == providerOrderID {
		return nil
	}
	if p.Status != domain.PaymentStatusPending {
		if p.Status == domain.PaymentStatusSuccess {
			return domain.ErrPaymentAlreadyComplete
		}
		if p.Status == domain.PaymentStatusRefundRequired {
			return domain.ErrPaymentRefundPending
		}
		return domain.ErrInvalidPayment
	}
	if p.ProviderOrderID != nil && *p.ProviderOrderID != providerOrderID {
		return domain.ErrInvalidPayment
	}
	p.ProviderOrderID = &providerOrderID
	return nil
}

func (m *mockPaymentRepo) FindAdminPayments(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminPaymentListResult, error) {
	return &domain.AdminPaymentListResult{
		Payments:     []domain.AdminPaymentListItem{},
		TotalRecords: 0,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   0,
	}, nil
}

func (m *mockPaymentRepo) GetEventPaymentSummaries(
	ctx context.Context,
	filter domain.AdminPaymentFilter,
) (*domain.AdminEventPaymentSummaryResult, error) {
	return &domain.AdminEventPaymentSummaryResult{
		Summaries:    []domain.AdminEventPaymentSummary{},
		TotalRecords: 0,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   0,
	}, nil
}

type mockBookingRepo struct {
	mu           sync.Mutex
	bookings     map[uuid.UUID]*domain.Booking
	items        map[uuid.UUID][]domain.BookingItem
	confirmCalls int
}

func newMockBookingRepo() *mockBookingRepo {
	return &mockBookingRepo{
		bookings: make(map[uuid.UUID]*domain.Booking),
		items:    make(map[uuid.UUID][]domain.BookingItem),
	}
}

func (m *mockBookingRepo) Create(ctx context.Context, b *domain.Booking, items []domain.BookingItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *b
	m.bookings[b.ID] = &copied
	m.items[b.ID] = items
	return nil
}

func (m *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.bookings[id]; ok {
		copied := *b
		return &copied, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return m.FindByID(ctx, id)
}

func (m *mockBookingRepo) FindByBookingNumber(ctx context.Context, num string) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.bookings {
		if b.BookingNumber == num {
			copied := *b
			return &copied, nil
		}
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.bookings {
		if b.ReservationID != nil && *b.ReservationID == resID {
			copied := *b
			return &copied, nil
		}
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.bookings[id]; ok {
		b.Status = status
		return nil
	}
	return domain.ErrBookingNotFound
}

func (m *mockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.confirmCalls++
	if b, ok := m.bookings[id]; ok {
		if b.Status == domain.BookingStatusPending {
			b.Status = domain.BookingStatusConfirmed
			return nil
		}
		return domain.ErrBookingNotFound
	}
	return domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindItemsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.BookingItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[bookingID], nil
}

func (m *mockBookingRepo) FindItemsByBookingIDs(ctx context.Context, bookingIDs []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[uuid.UUID][]domain.BookingItem)
	for _, id := range bookingIDs {
		if items, ok := m.items[id]; ok {
			res[id] = items
		}
	}
	return res, nil
}

func (m *mockBookingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Booking
	for _, b := range m.bookings {
		if b.UserID == userID {
			res = append(res, *b)
		}
	}
	return res, nil
}

func (m *mockBookingRepo) FindByUserIDPaginated(ctx context.Context, userID uuid.UUID, filter domain.CustomerBookingFilter) ([]domain.Booking, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Booking
	for _, b := range m.bookings {
		if b.UserID == userID {
			res = append(res, *b)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockBookingRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[id]
	if !ok {
		return nil, nil, domain.ErrBookingNotFound
	}
	items := m.items[id]
	copied := *b
	return &copied, items, nil
}

func (m *mockBookingRepo) CountConfirmedByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, b := range m.bookings {
		if b.EventID == eventID && b.Status == domain.BookingStatusConfirmed {
			count++
		}
	}
	return count, nil
}

func (m *mockBookingRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, b := range m.bookings {
		if b.EventID == eventID {
			count++
		}
	}
	return count, nil
}

func (m *mockBookingRepo) GetBookingStatsByEventID(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}

func (m *mockBookingRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Booking
	for _, b := range m.bookings {
		if b.EventID == eventID {
			res = append(res, *b)
		}
	}
	return res, nil
}

type mockCheckoutRepo struct {
	mu        sync.Mutex
	checkouts map[uuid.UUID]*domain.Checkout
}

func newMockCheckoutRepo() *mockCheckoutRepo {
	return &mockCheckoutRepo{
		checkouts: make(map[uuid.UUID]*domain.Checkout),
	}
}

func (m *mockCheckoutRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Checkout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.checkouts[id]; ok {
		copied := *c
		return &copied, nil
	}
	return nil, domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Checkout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.checkouts {
		if c.ReservationID == resID {
			copied := *c
			return &copied, nil
		}
	}
	return nil, domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.checkouts[id]; ok {
		c.Status = status
		return nil
	}
	return domain.ErrCheckoutNotFound
}

func (m *mockCheckoutRepo) Create(ctx context.Context, c *domain.Checkout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *c
	m.checkouts[c.ID] = &copied
	return nil
}

func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, resID uuid.UUID) (bool, error) {
	return false, nil
}

type mockSeatReservationRepo struct {
	mu           sync.Mutex
	reservations map[uuid.UUID]*domain.SeatReservation
	items        map[uuid.UUID][]domain.SeatReservationItem
	cancelErr    error
}

func newMockSeatReservationRepo() *mockSeatReservationRepo {
	return &mockSeatReservationRepo{
		reservations: make(map[uuid.UUID]*domain.SeatReservation),
		items:        make(map[uuid.UUID][]domain.SeatReservationItem),
	}
}

func (m *mockSeatReservationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		copied := *r
		return &copied, nil
	}
	return nil, domain.ErrSeatReservationNotFound
}

func (m *mockSeatReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	return m.FindByID(ctx, id)
}

func (m *mockSeatReservationRepo) CompleteIfActive(ctx context.Context, id uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		if r.Status == domain.ReservationStatusActive && time.Now().Before(r.ExpiresAt) {
			r.Status = domain.ReservationStatusCompleted
			return true, nil
		}
		return false, nil
	}
	return false, domain.ErrSeatReservationNotFound
}

func (m *mockSeatReservationRepo) CancelIfActive(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancelErr != nil {
		return false, m.cancelErr
	}
	if r, ok := m.reservations[id]; ok {
		if r.Status == domain.ReservationStatusActive {
			r.Status = domain.ReservationStatusCancelled
			return true, nil
		}
	}
	return false, nil
}

func (m *mockSeatReservationRepo) ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return true, nil
}

func (m *mockSeatReservationRepo) FindExpiredActive(ctx context.Context, now time.Time, limit int) ([]domain.SeatReservation, error) {
	return nil, nil
}

func (m *mockSeatReservationRepo) FindItemsByReservationID(ctx context.Context, id uuid.UUID) ([]domain.SeatReservationItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[id], nil
}

func (m *mockSeatReservationRepo) Create(ctx context.Context, r *domain.SeatReservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *r
	m.reservations[r.ID] = &copied
	return nil
}

func (m *mockSeatReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ReservationID] = append(m.items[item.ReservationID], *item)
	return nil
}

func (m *mockSeatReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		r.Status = status
		return nil
	}
	return domain.ErrSeatReservationNotFound
}

func (m *mockSeatReservationRepo) CountActiveByEventID(ctx context.Context, eventID uuid.UUID, now time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, r := range m.reservations {
		if r.EventID == eventID && r.Status == domain.ReservationStatusActive && now.Before(r.ExpiresAt) {
			count++
		}
	}
	return count, nil
}

type mockTicketRepo struct {
	mu      sync.Mutex
	tickets []domain.Ticket
}

func (m *mockTicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets = append(m.tickets, *ticket)
	return nil
}

func (m *mockTicketRepo) CreateMany(ctx context.Context, tickets []domain.Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets = append(m.tickets, tickets...)
	return nil
}

func (m *mockTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Ticket
	for _, t := range m.tickets {
		if t.BookingID == bookingID {
			list = append(list, t)
		}
	}
	return list, nil
}
func (m *mockTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.tickets {
		if m.tickets[i].BookingID == bookingID {
			m.tickets[i].Status = status
		}
	}
	return nil
}
func (m *mockTicketRepo) UpdateStatusForTicketIDs(ctx context.Context, ticketIDs []uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idSet := make(map[uuid.UUID]struct{}, len(ticketIDs))
	for _, id := range ticketIDs {
		idSet[id] = struct{}{}
	}
	for i := range m.tickets {
		if _, ok := idSet[m.tickets[i].ID]; ok {
			m.tickets[i].Status = status
		}
	}
	return nil
}
func (m *mockTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockTicketRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Ticket
	for _, t := range m.tickets {
		if t.EventID == eventID {
			res = append(res, t)
		}
	}
	return res, nil
}

type mockSeatRepo struct {
	domain.SeatRepository
	releasedSeats []uuid.UUID
	releaseErr    error
	mu            sync.Mutex
}

func (m *mockSeatRepo) BookReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	return nil
}

func (m *mockSeatRepo) ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.releaseErr != nil {
		return m.releaseErr
	}
	m.releasedSeats = append(m.releasedSeats, seatID)
	return nil
}

func (m *mockSeatRepo) FindDetailsBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (map[uuid.UUID]domain.SeatDetail, error) {
	return make(map[uuid.UUID]domain.SeatDetail), nil
}

type mockUserRepo struct {
	domain.UserRepository
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: id, FullName: "Test User", Email: "test@example.com"}, nil
}

type mockEventRepo struct {
	domain.EventRepository
	organizerID uuid.UUID
	status      string
	findErr     error
}

func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	orgID := m.organizerID
	if orgID == uuid.Nil {
		orgID = uuid.New()
	}
	st := m.status
	if st == "" {
		st = domain.EventStatusPublished
	}
	return &domain.Event{ID: id, OrganizerID: orgID, Title: "Test Event", Status: st, EventType: domain.EventTypePhysical}, nil
}

type mockEventScheduleRepo struct {
	domain.EventScheduleRepository
}

func (m *mockEventScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	return &domain.EventSchedule{EventID: eventID, EventDate: time.Now()}, nil
}

type mockVenueRepo struct {
	domain.VenueRepository
}

func (m *mockVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	return &domain.Venue{ID: id, Name: "Test Venue"}, nil
}

type mockOrganizerProfileRepo struct {
	domain.OrganizerProfileRepository
}

func (m *mockOrganizerProfileRepo) FindByOrganizerID(id uuid.UUID) (*domain.OrganizerProfile, error) {
	return &domain.OrganizerProfile{OrganizerID: id, BusinessName: "Test Organizer"}, nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	releasedQuantities map[uuid.UUID]int
	releaseErr         error
	mu                 sync.Mutex
}

func (m *mockTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	return &domain.TicketType{ID: id, Name: "General Admission"}, nil
}

func (m *mockTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ticketTypeID uuid.UUID, quantity int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.releaseErr != nil {
		return m.releaseErr
	}
	if m.releasedQuantities == nil {
		m.releasedQuantities = make(map[uuid.UUID]int)
	}
	m.releasedQuantities[ticketTypeID] += quantity
	return nil
}

type mockFinanceUsecase struct {
	mu                      sync.Mutex
	recordCaptureWithinTxFn func(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordPaymentCaptureInput) (*domainfinance.JournalTransaction, error)
	recordCaptureFn         func(ctx context.Context, in domain.RecordPaymentCaptureInput) (*domainfinance.JournalTransaction, error)
	recordRefundWithinTxFn  func(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error)
	recordRefundFn          func(ctx context.Context, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error)
	recordEscrowWithinTxFn  func(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordEscrowCaptureInput) (*domainfinance.JournalTransaction, error)
	recordEscrowFn          func(ctx context.Context, in domain.RecordEscrowCaptureInput) (*domainfinance.JournalTransaction, error)
	capturedInputs          []domain.RecordPaymentCaptureInput
	refundInputs            []domain.RecordRefundInput
	escrowInputs            []domain.RecordEscrowCaptureInput
}

func (m *mockFinanceUsecase) RecordPaymentCaptureWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordPaymentCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordCaptureWithinTxFn != nil {
		return m.recordCaptureWithinTxFn(ctx, txRepos, in)
	}
	m.capturedInputs = append(m.capturedInputs, in)
	return nil, nil
}

func (m *mockFinanceUsecase) RecordPaymentCapture(
	ctx context.Context,
	in domain.RecordPaymentCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordCaptureFn != nil {
		return m.recordCaptureFn(ctx, in)
	}
	m.capturedInputs = append(m.capturedInputs, in)
	return nil, nil
}

func (m *mockFinanceUsecase) RecordRefundWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordRefundInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordRefundWithinTxFn != nil {
		return m.recordRefundWithinTxFn(ctx, txRepos, in)
	}
	m.refundInputs = append(m.refundInputs, in)
	return nil, nil
}

func (m *mockFinanceUsecase) RecordRefund(
	ctx context.Context,
	in domain.RecordRefundInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordRefundFn != nil {
		return m.recordRefundFn(ctx, in)
	}
	m.refundInputs = append(m.refundInputs, in)
	return nil, nil
}

func (m *mockFinanceUsecase) RecordEscrowCaptureWithinTx(
	ctx context.Context,
	txRepos domain.TransactionRepositories,
	in domain.RecordEscrowCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordEscrowWithinTxFn != nil {
		return m.recordEscrowWithinTxFn(ctx, txRepos, in)
	}
	m.escrowInputs = append(m.escrowInputs, in)
	return nil, nil
}

func (m *mockFinanceUsecase) RecordEscrowCapture(
	ctx context.Context,
	in domain.RecordEscrowCaptureInput,
) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordEscrowFn != nil {
		return m.recordEscrowFn(ctx, in)
	}
	m.escrowInputs = append(m.escrowInputs, in)
	return nil, nil
}

type mockInvoiceRepo struct {
	domain.InvoiceRepository
	invoicesByBooking map[uuid.UUID]*domain.Invoice
	seq               int64
	mu                sync.Mutex
	createErr         error
	findByBookingErr  error
	nextNumberErr     error
}

func newMockInvoiceRepo() *mockInvoiceRepo {
	return &mockInvoiceRepo{
		invoicesByBooking: make(map[uuid.UUID]*domain.Invoice),
	}
}

func (m *mockInvoiceRepo) NextReceiptNumber(ctx context.Context, t time.Time) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nextNumberErr != nil {
		return "", m.nextNumberErr
	}
	m.seq++
	return fmt.Sprintf("EVH-REC-%s-%06d", t.Format("20060102"), m.seq), nil
}

func (m *mockInvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice, items []domain.InvoiceItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.invoicesByBooking[invoice.BookingID]; exists {
		return domain.ErrInvoiceAlreadyExists
	}
	invoice.Items = items
	m.invoicesByBooking[invoice.BookingID] = invoice
	return nil
}

func (m *mockInvoiceRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*domain.Invoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByBookingErr != nil {
		return nil, m.findByBookingErr
	}
	inv, ok := m.invoicesByBooking[bookingID]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	return inv, nil
}

func (m *mockInvoiceRepo) FindByBookingIDWithItems(ctx context.Context, bookingID uuid.UUID) (*domain.Invoice, []domain.InvoiceItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invoicesByBooking[bookingID]
	if !ok {
		return nil, nil, domain.ErrInvoiceNotFound
	}
	return inv, inv.Items, nil
}

type mockRazorpayWebhookEventRepo struct {
	mu     sync.Mutex
	events map[string]string
}

func (m *mockRazorpayWebhookEventRepo) Create(ctx context.Context, eventID string, eventType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.events[eventID]; ok {
		return domain.ErrDuplicateWebhookEvent
	}
	m.events[eventID] = eventType
	return nil
}

type mockEventCancellationRepo struct {
	domain.EventCancellationRepository
	cancellations map[uuid.UUID]*domain.EventCancellation
}

func (m *mockEventCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	if m.cancellations != nil {
		if c, ok := m.cancellations[eventID]; ok {
			return c, nil
		}
	}
	return nil, domain.ErrEventNotFound
}

type mockTxRepos struct {
	domain.TransactionRepositories
	paymentRepo            *mockPaymentRepo
	bookingRepo            *mockBookingRepo
	checkoutRepo           *mockCheckoutRepo
	seatReservationRepo    *mockSeatReservationRepo
	ticketRepo             *mockTicketRepo
	seatRepo               *mockSeatRepo
	ticketTypeRepo         *mockTicketTypeRepo
	webhookEventRepo       *mockRazorpayWebhookEventRepo
	eventCancellationRepo  *mockEventCancellationRepo
	eventRepo              *mockEventRepo
	invoiceRepo            *mockInvoiceRepo
	refundRepo             *mockRefundRepo
	executeWithSavepointFn func(name domain.SavepointName, fn func() error) (domain.SavepointResult, error)
}

func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository {
	return m.paymentRepo
}
func (m *mockTxRepos) BookingRepository() domain.BookingRepository {
	return m.bookingRepo
}
func (m *mockTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return m.checkoutRepo
}
func (m *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return m.seatReservationRepo
}
func (m *mockTxRepos) TicketRepository() domain.TicketRepository {
	return m.ticketRepo
}
func (m *mockTxRepos) SeatRepository() domain.SeatRepository {
	return m.seatRepo
}
func (m *mockTxRepos) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository {
	return m.webhookEventRepo
}
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.eventCancellationRepo
}
func (m *mockTxRepos) RefundRepository() domain.RefundRepository {
	if m.refundRepo == nil {
		m.refundRepo = newMockRefundRepo()
	}
	return m.refundRepo
}
func (m *mockTxRepos) UserRepository() domain.UserRepository {
	return &mockUserRepo{}
}
func (m *mockTxRepos) EventRepository() domain.EventRepository {
	if m.eventRepo == nil {
		m.eventRepo = &mockEventRepo{}
	}
	return m.eventRepo
}
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return &mockEventScheduleRepo{}
}
func (m *mockTxRepos) VenueRepository() domain.VenueRepository {
	return &mockVenueRepo{}
}
func (m *mockTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return &mockOrganizerProfileRepo{}
}
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	if m.ticketTypeRepo == nil {
		m.ticketTypeRepo = &mockTicketTypeRepo{}
	}
	return m.ticketTypeRepo
}
func (m *mockTxRepos) InvoiceRepository() domain.InvoiceRepository {
	if m.invoiceRepo == nil {
		m.invoiceRepo = newMockInvoiceRepo()
	}
	return m.invoiceRepo
}

func (m *mockTxRepos) ExecuteWithSavepoint(name domain.SavepointName, fn func() error) (domain.SavepointResult, error) {
	if m.executeWithSavepointFn != nil {
		return m.executeWithSavepointFn(name, fn)
	}
	if fn == nil {
		return domain.SavepointResult{}, domain.ErrNilSavepointCallback
	}
	if name != domain.SavepointTaxRefundInventory {
		return domain.SavepointResult{}, domain.ErrInvalidSavepointName
	}
	fnErr := fn()
	if fnErr != nil {
		return domain.SavepointResult{
			Status:      domain.SavepointStatusRolledBack,
			CallbackErr: fnErr,
		}, nil
	}
	return domain.SavepointResult{
		Status:      domain.SavepointStatusCommitted,
		CallbackErr: nil,
	}, nil
}

type mockRefundRepo struct {
	domain.RefundRepository
	mu                 sync.Mutex
	refundsByID        map[uuid.UUID]*domain.Refund
	refundsByGatewayID map[string]*domain.Refund
	updateCalls        int
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{
		refundsByID:        make(map[uuid.UUID]*domain.Refund),
		refundsByGatewayID: make(map[string]*domain.Refund),
	}
}

func (m *mockRefundRepo) Create(ctx context.Context, r *domain.Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *r
	m.refundsByID[r.ID] = &copied
	if r.GatewayRefundID != nil {
		m.refundsByGatewayID[*r.GatewayRefundID] = &copied
	}
	return nil
}

func (m *mockRefundRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.refundsByID[id]; ok {
		copied := *r
		return &copied, nil
	}
	return nil, domain.ErrRefundNotFound
}

func (m *mockRefundRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	return m.FindByID(ctx, id)
}

func (m *mockRefundRepo) FindByGatewayRefundID(ctx context.Context, gatewayRefundID string) (*domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.refundsByGatewayID[gatewayRefundID]; ok {
		copied := *r
		return &copied, nil
	}
	return nil, domain.ErrRefundNotFound
}

func (m *mockRefundRepo) FindByGatewayRefundIDForUpdate(ctx context.Context, gatewayRefundID string) (*domain.Refund, error) {
	return m.FindByGatewayRefundID(ctx, gatewayRefundID)
}

func (m *mockRefundRepo) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []domain.Refund
	for _, r := range m.refundsByID {
		if r.PaymentID == paymentID {
			results = append(results, *r)
		}
	}
	return results, nil
}

func (m *mockRefundRepo) Update(ctx context.Context, r *domain.Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	copied := *r
	m.refundsByID[r.ID] = &copied
	if r.GatewayRefundID != nil {
		m.refundsByGatewayID[*r.GatewayRefundID] = &copied
	}
	return nil
}

type mockTransactionManager struct {
	repos *mockTxRepos
	mu    sync.Mutex
}

func (m *mockTransactionManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(m.repos)
}

type mockPaymentProvider struct {
	mu              sync.Mutex
	initiateCalls   int
	ordersByReceipt map[string]*payment.ProviderOrder
	orderStatusMap  map[string]*payment.ProviderOrder
	initiateErr     error
	verifyResult    bool
}

func newMockPaymentProvider() *mockPaymentProvider {
	return &mockPaymentProvider{
		ordersByReceipt: make(map[string]*payment.ProviderOrder),
		orderStatusMap:  make(map[string]*payment.ProviderOrder),
		verifyResult:    true,
	}
}

func (m *mockPaymentProvider) Initiate(ctx context.Context, amount decimal.Decimal, currency string, idempotencyKey string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initiateCalls++
	if m.initiateErr != nil {
		return "", m.initiateErr
	}
	orderID := "order_" + uuid.NewString()[:8]
	order := &payment.ProviderOrder{
		ID:       orderID,
		Status:   payment.OrderStatusUsable,
		Amount:   amount,
		Currency: currency,
	}
	m.ordersByReceipt[idempotencyKey] = order
	m.orderStatusMap[orderID] = order
	return orderID, nil
}

func (m *mockPaymentProvider) Verify(ctx context.Context, providerOrderID string, paymentID string, signature string, expectedAmount decimal.Decimal, currency string) (bool, error) {
	return m.verifyResult, nil
}

func (m *mockPaymentProvider) VerifyWebhookSignature(body []byte, signature string) bool {
	return true
}

func (m *mockPaymentProvider) FindOrderByReceipt(ctx context.Context, receipt string, expectedAmount decimal.Decimal, currency string) (*payment.ProviderOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if order, ok := m.ordersByReceipt[receipt]; ok {
		return order, nil
	}
	return nil, nil // not found
}

func (m *mockPaymentProvider) CheckOrderStatus(ctx context.Context, providerOrderID string, expectedAmount decimal.Decimal, currency string) (*payment.ProviderOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if order, ok := m.orderStatusMap[providerOrderID]; ok {
		return order, nil
	}
	return nil, errors.New("order not found")
}

func (m *mockPaymentProvider) Refund(ctx context.Context, gatewayPaymentID string, amount decimal.Decimal, currency string, idempotencyKey string) (*domain.RefundResult, error) {
	return &domain.RefundResult{
		RefundID: "rfnd_" + uuid.NewString()[:8],
		Amount:   amount,
		Status:   domain.RefundStatusSuccess,
	}, nil
}

func (m *mockPaymentProvider) FetchRefunds(ctx context.Context, gatewayPaymentID string) ([]domain.RefundResult, error) {
	return []domain.RefundResult{}, nil
}

// --- Test Fixture Setup ---

type testFixture struct {
	usecase         *payment.PaymentUsecase
	paymentRepo     *mockPaymentRepo
	bookingRepo     *mockBookingRepo
	checkoutRepo    *mockCheckoutRepo
	reservationRepo *mockSeatReservationRepo
	ticketRepo      *mockTicketRepo
	seatRepo        *mockSeatRepo
	ticketTypeRepo  *mockTicketTypeRepo
	invoiceRepo     *mockInvoiceRepo
	refundRepo      *mockRefundRepo
	financeUsecase  *mockFinanceUsecase
	txRepos         *mockTxRepos
	provider        *mockPaymentProvider
	userID          uuid.UUID
	checkoutID      uuid.UUID
	reservationID   uuid.UUID
	eventID         uuid.UUID
	totalAmount     decimal.Decimal
}

func setupFixture(t *testing.T) *testFixture {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	paymentRepo := newMockPaymentRepo()
	bookingRepo := newMockBookingRepo()
	checkoutRepo := newMockCheckoutRepo()
	reservationRepo := newMockSeatReservationRepo()
	ticketRepo := &mockTicketRepo{}
	seatRepo := &mockSeatRepo{}
	ticketTypeRepo := &mockTicketTypeRepo{releasedQuantities: make(map[uuid.UUID]int)}
	eventRepo := &mockEventRepo{organizerID: uuid.New()}
	invoiceRepo := newMockInvoiceRepo()
	refundRepo := newMockRefundRepo()
	webhookEventRepo := &mockRazorpayWebhookEventRepo{events: make(map[string]string)}
	financeUC := &mockFinanceUsecase{}

	txRepos := &mockTxRepos{
		paymentRepo:           paymentRepo,
		bookingRepo:           bookingRepo,
		checkoutRepo:          checkoutRepo,
		seatReservationRepo:   reservationRepo,
		ticketRepo:            ticketRepo,
		seatRepo:              seatRepo,
		ticketTypeRepo:        ticketTypeRepo,
		eventRepo:              eventRepo,
		invoiceRepo:           invoiceRepo,
		refundRepo:            refundRepo,
		webhookEventRepo:      webhookEventRepo,
		eventCancellationRepo: &mockEventCancellationRepo{cancellations: make(map[uuid.UUID]*domain.EventCancellation)},
	}
	tm := &mockTransactionManager{repos: txRepos}

	provider := newMockPaymentProvider()
	bookingUC := bookinguc.NewBookingUsecase(tm, provider, logger)
	paymentUC, err := payment.NewPaymentUsecase(tm, bookingUC, financeUC, provider, logger)
	require.NoError(t, err)

	userID := uuid.New()
	eventID := uuid.New()
	resID := uuid.New()
	checkoutID := uuid.New()
	amount := decimal.NewFromInt(500)

	// Create valid seat reservation
	now := time.Now()
	res := &domain.SeatReservation{
		ID:        resID,
		UserID:    userID,
		EventID:   eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(15 * time.Minute),
	}
	require.NoError(t, reservationRepo.Create(context.Background(), res))
	seatID := uuid.New()
	require.NoError(t, reservationRepo.CreateItem(context.Background(), &domain.SeatReservationItem{
		ReservationID: resID,
		SeatID:        &seatID,
		Quantity:      1,
		Price:         amount,
	}))

	// Create valid checkout
	checkout := &domain.Checkout{
		ID:            checkoutID,
		UserID:        userID,
		ReservationID: resID,
		TotalAmount:   amount,
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     now.Add(15 * time.Minute),
	}
	require.NoError(t, checkoutRepo.Create(context.Background(), checkout))

	return &testFixture{
		usecase:         paymentUC,
		paymentRepo:     paymentRepo,
		bookingRepo:     bookingRepo,
		checkoutRepo:    checkoutRepo,
		reservationRepo: reservationRepo,
		ticketRepo:      ticketRepo,
		seatRepo:        seatRepo,
		ticketTypeRepo:  ticketTypeRepo,
		invoiceRepo:     invoiceRepo,
		refundRepo:      refundRepo,
		financeUsecase:  financeUC,
		txRepos:         txRepos,
		provider:        provider,
		userID:          userID,
		checkoutID:      checkoutID,
		reservationID:   resID,
		eventID:         eventID,
		totalAmount:     amount,
	}
}

// --- Option C Business Rules Tests ---

func TestCreatePayment_OptionC_Rule1_ReuseUsableOrder(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// 1st attempt creates payment and order
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)
	require.NotNil(t, p1)
	require.NotNil(t, p1.ProviderOrderID)
	orderID := *p1.ProviderOrderID
	assert.Equal(t, 1, f.provider.initiateCalls)

	// 2nd attempt must reuse existing pending payment and provider order
	p2, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)
	require.NotNil(t, p2)
	assert.Equal(t, p1.ID, p2.ID, "Must reuse identical payment ID")
	assert.Equal(t, orderID, *p2.ProviderOrderID, "Must reuse identical provider order ID")
	assert.Equal(t, 1, f.provider.initiateCalls, "Must NOT call Initiate again")
}

func TestCreatePayment_OptionC_Rule2_RetireUnusableOrder(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// 1st attempt creates payment
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)
	oldOrderID := *p1.ProviderOrderID

	// Simulate order becoming UNUSABLE on gateway (e.g. expired on gateway or amount mismatch)
	f.provider.orderStatusMap[oldOrderID].Status = payment.OrderStatusUnusable

	// 2nd attempt should retire p1 as FAILED and create a new attempt
	p2, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)
	assert.NotEqual(t, p1.ID, p2.ID, "Must generate a new payment ID")
	assert.NotEqual(t, oldOrderID, *p2.ProviderOrderID, "Must generate a new provider order ID")
	assert.Equal(t, 2, f.provider.initiateCalls, "Must call Initiate for replacement attempt")

	// Verify p1 was retired as FAILED
	retired, findErr := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, findErr)
	assert.Equal(t, domain.PaymentStatusFailed, retired.Status)
}

func TestCreatePayment_OptionC_Rule3_RetryAfterFailedPayments(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// Create initial booking
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)

	// Simulate payment failed via webhook or gateway rejection
	require.NoError(t, f.paymentRepo.MarkFailed(ctx, p1.ID))

	// Next attempt must be allowed (Rule 3)
	p2, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)
	assert.NotEqual(t, p1.ID, p2.ID)
	assert.Equal(t, domain.PaymentStatusPending, p2.Status)
}

func TestCreatePayment_OptionC_Rule4_SuccessBlocksNewAttempts(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)

	// Mark payment successful
	now := time.Now()
	require.NoError(t, f.paymentRepo.MarkSuccessful(ctx, p1.ID, "pay_12345", now))

	// Further attempts must be rejected
	_, err = f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	assert.ErrorIs(t, err, domain.ErrPaymentAlreadyComplete)
}

func TestCreatePayment_OptionC_Rule5_RefundRequiredBlocksNewAttempts(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	require.NoError(t, err)

	// Mark payment REFUND_REQUIRED
	now := time.Now()
	require.NoError(t, f.paymentRepo.MarkRefundRequired(ctx, p1.ID, "pay_refund_123", now))

	// Further attempts must be rejected with ErrPaymentRefundPending
	_, err = f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodUPI,
	})
	assert.ErrorIs(t, err, domain.ErrPaymentRefundPending)
}

// --- Razorpay Receipt Reconciliation Tests ---

func TestCreatePayment_ReceiptReconciliation_OrderRecovered(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// Initial attempt: gateway initiate fails with a timeout
	f.provider.initiateErr = errors.New("gateway connection timeout")

	_, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	assert.Error(t, err)

	// Verify a pending payment exists in DB with provider_order_id == nil
	booking, err := f.bookingRepo.FindByReservationID(ctx, f.reservationID)
	require.NoError(t, err)
	payments, err := f.paymentRepo.FindByBookingID(ctx, booking.ID.String())
	require.NoError(t, err)
	require.Len(t, payments, 1)
	p := payments[0]
	assert.Nil(t, p.ProviderOrderID, "Must have nil provider_order_id after timeout")

	// Now simulate gateway having registered the order under payment.ID (receipt)
	gatewayOrderID := "order_recovered_123"
	recoveredOrder := &payment.ProviderOrder{
		ID:       gatewayOrderID,
		Status:   payment.OrderStatusUsable,
		Amount:   f.totalAmount,
		Currency: "INR",
	}
	f.provider.ordersByReceipt[p.ID] = recoveredOrder
	f.provider.orderStatusMap[gatewayOrderID] = recoveredOrder
	f.provider.initiateErr = nil // gateway recovered

	// Second CreatePayment call should reconcile by receipt, recover gatewayOrderID, and succeed
	result, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, p.ID, result.ID, "Must reconcile and reuse same payment")
	assert.NotNil(t, result.ProviderOrderID)
	assert.Equal(t, gatewayOrderID, *result.ProviderOrderID, "Must recover order ID from receipt lookup")
}

func TestCreatePayment_ReceiptReconciliation_OrderAbsent_RetriesWithSameReceipt(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// Initial attempt: gateway initiate fails with a timeout
	f.provider.initiateErr = errors.New("gateway connection timeout")

	_, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	assert.Error(t, err)

	booking, err := f.bookingRepo.FindByReservationID(ctx, f.reservationID)
	require.NoError(t, err)
	payments, err := f.paymentRepo.FindByBookingID(ctx, booking.ID.String())
	require.NoError(t, err)
	require.Len(t, payments, 1)
	p := payments[0]

	// Receipt lookup returns 0 orders (f.provider.ordersByReceipt is empty for p.ID)
	// Gateway is now healthy again
	f.provider.initiateErr = nil

	// Second attempt should retry with the SAME payment.ID and succeed without creating a second payment record
	result, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, p.ID, result.ID, "Must reuse same payment ID as receipt")
	assert.NotNil(t, result.ProviderOrderID)

	// Verify only 1 payment record exists in DB for this booking
	allPayments, err := f.paymentRepo.FindByBookingID(ctx, booking.ID.String())
	require.NoError(t, err)
	assert.Len(t, allPayments, 1, "Must never create a duplicate payment row")
}

func TestCreatePayment_OrderStatus_InFlightFailsClosed(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	// Set gateway status to IN_FLIGHT
	f.provider.orderStatusMap[*p1.ProviderOrderID].Status = payment.OrderStatusInFlight

	// Next call must fail closed without retiring the payment
	_, err = f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	assert.ErrorIs(t, err, domain.ErrPaymentInProgress)

	// Payment must remain PENDING
	current, findErr := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, findErr)
	assert.Equal(t, domain.PaymentStatusPending, current.Status)
}

func TestCreatePayment_OrderStatus_TransientErrorFailsClosed(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	// Remove from provider orderStatusMap to simulate gateway 500 / error
	delete(f.provider.orderStatusMap, *p1.ProviderOrderID)

	// Next call must fail closed with ErrPaymentProviderUnavailable
	_, err = f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	assert.ErrorIs(t, err, domain.ErrPaymentProviderUnavailable)

	// Local payment status must remain PENDING (never retired on transient failure)
	current, findErr := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, findErr)
	assert.Equal(t, domain.PaymentStatusPending, current.Status)
}

// --- Concurrency Tests ---

func TestCreatePayment_ConcurrentCalls_SingleOrderCreated(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	const concurrency = 10
	var wg sync.WaitGroup
	results := make([]*domain.Payment, concurrency)
	errorsList := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errorsList[idx] = f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
				CheckoutID: f.checkoutID,
				UserID:     f.userID,
				Method:     domain.PaymentMethodUPI,
			})
		}(i)
	}

	wg.Wait()

	// Exactly one order initiated, or all successful ones shared the exact same order ID
	var successCount int
	var sharedOrderID string
	for i := 0; i < concurrency; i++ {
		if errorsList[i] == nil && results[i] != nil {
			successCount++
			if sharedOrderID == "" {
				sharedOrderID = *results[i].ProviderOrderID
			} else {
				assert.Equal(t, sharedOrderID, *results[i].ProviderOrderID, "All concurrent requests must share order ID")
			}
		}
	}

	assert.Positive(t, successCount)
	assert.Equal(t, 1, f.provider.initiateCalls, "Only 1 external order must be initiated despite 10 concurrent requests")
}

// --- Duplicate Capture & Booking Integrity Tests ---

func TestFinalizeBookingPayment_DuplicateCapturePreservesConfirmedBooking(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// 1. Initial successful payment and verification
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	verifyRes, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_initial_123",
		RazorpaySignature: "sig_dummy",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusConfirmed, verifyRes.Booking.Status)
	assert.Equal(t, 1, len(f.ticketRepo.tickets))

	// 2. Simulate a SECOND payment attempt was also captured at the gateway
	order2ID := "order_secondary_456"
	p2 := &domain.Payment{
		ID:              uuid.New().String(),
		BookingID:       verifyRes.Booking.ID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &order2ID,
		Method:          domain.PaymentMethodUPI,
		Amount:          f.totalAmount,
		Status:          domain.PaymentStatusPending,
	}
	require.NoError(t, f.paymentRepo.Create(ctx, p2))

	// Process webhook for second captured payment
	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_duplicate_capture_001",
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "pay_duplicate_456",
					"order_id": "%s",
					"status": "captured",
					"amount": %d,
					"currency": "INR"
				}
			}
		}
	}`, order2ID, f.totalAmount.Mul(decimal.NewFromInt(100)).IntPart()))

	err = f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_duplicate_capture_001")
	require.NoError(t, err)

	// Booking MUST remain CONFIRMED
	booking, findErr := f.bookingRepo.FindByID(ctx, verifyRes.Booking.ID)
	require.NoError(t, findErr)
	assert.Equal(t, domain.BookingStatusConfirmed, booking.Status, "Confirmed booking must NEVER be cancelled by secondary capture")

	// Secondary payment must be flagged REFUND_REQUIRED
	refundedPayment, findPErr := f.paymentRepo.FindByID(ctx, p2.ID)
	require.NoError(t, findPErr)
	assert.Equal(t, domain.PaymentStatusRefundRequired, refundedPayment.Status)
	assert.Equal(t, "pay_duplicate_456", *refundedPayment.TransactionID)

	// Tickets must NOT be duplicated
	tickets, ticketErr := f.ticketRepo.FindByBookingID(ctx, verifyRes.Booking.ID)
	require.NoError(t, ticketErr)
	assert.Equal(t, 1, len(tickets), "Tickets must not be generated again for duplicate capture")
}

func TestMarkRefundRequired_IdempotentFulfillment(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	now := time.Now()
	// First refund transition
	err = f.paymentRepo.MarkRefundRequired(ctx, p1.ID, "pay_tx_123", now)
	require.NoError(t, err)

	// Second refund transition with identical transaction ID must succeed idempotently
	err = f.paymentRepo.MarkRefundRequired(ctx, p1.ID, "pay_tx_123", now)
	require.NoError(t, err)

	// Transitioning with a conflicting transaction ID when already refund-required must be rejected
	err = f.paymentRepo.MarkRefundRequired(ctx, p1.ID, "pay_tx_conflicting", now)
	assert.Error(t, err)
}

func TestUpdateProviderOrderID_ZeroRowReconciliation(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	// Calling UpdateProviderOrderID with the already-persisted order ID must succeed idempotently
	err = f.paymentRepo.UpdateProviderOrderID(ctx, p1.ID, *p1.ProviderOrderID)
	require.NoError(t, err)

	// Calling with non-existent payment ID must return ErrPaymentNotFound
	err = f.paymentRepo.UpdateProviderOrderID(ctx, uuid.NewString(), "order_random")
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestNewPaymentUsecase_Validation(t *testing.T) {
	tm := &mockTransactionManager{}
	provider := newMockPaymentProvider()
	bookingUC := bookinguc.NewBookingUsecase(tm, provider, slog.Default())
	financeUC := &mockFinanceUsecase{}
	logger := slog.Default()

	t.Run("Rejects nil transactionManager", func(t *testing.T) {
		uc, err := payment.NewPaymentUsecase(nil, bookingUC, financeUC, provider, logger)
		require.Error(t, err)
		assert.Nil(t, uc)
		assert.Contains(t, err.Error(), "transactionManager cannot be nil")
	})

	t.Run("Rejects nil bookingUsecase", func(t *testing.T) {
		uc, err := payment.NewPaymentUsecase(tm, nil, financeUC, provider, logger)
		require.Error(t, err)
		assert.Nil(t, uc)
		assert.Contains(t, err.Error(), "bookingUsecase cannot be nil")
	})

	t.Run("Rejects nil financeUsecase", func(t *testing.T) {
		uc, err := payment.NewPaymentUsecase(tm, bookingUC, nil, provider, logger)
		require.Error(t, err)
		assert.Nil(t, uc)
		assert.Contains(t, err.Error(), "financeUsecase cannot be nil")
	})

	t.Run("Rejects nil provider", func(t *testing.T) {
		uc, err := payment.NewPaymentUsecase(tm, bookingUC, financeUC, nil, logger)
		require.Error(t, err)
		assert.Nil(t, uc)
		assert.Contains(t, err.Error(), "provider cannot be nil")
	})

	t.Run("Accepts valid dependencies and defaults nil logger", func(t *testing.T) {
		uc, err := payment.NewPaymentUsecase(tm, bookingUC, financeUC, provider, nil)
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})
}

func TestVerifyPayment_NormalPaymentCaptureCreatesFinancialJournal(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	verifyRes, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_normal_123",
		RazorpaySignature: "sig_dummy",
	})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	assert.Equal(t, domain.PaymentStatusSuccess, verifyRes.Payment.Status)
	assert.Equal(t, domain.BookingStatusConfirmed, verifyRes.Booking.Status)

	// Verify that financeUsecase recorded the payment capture
	require.Len(t, f.financeUsecase.capturedInputs, 1)
	captured := f.financeUsecase.capturedInputs[0]
	assert.Equal(t, p1.ID, captured.PaymentID.String())
	assert.Equal(t, verifyRes.Booking.ID, captured.BookingID)
	assert.True(t, verifyRes.Booking.Total.Equal(captured.Total))
	assert.True(t, verifyRes.Booking.Subtotal.Equal(captured.Subtotal))
	assert.True(t, verifyRes.Booking.Tax.Equal(captured.Tax))
	assert.Equal(t, "INR", captured.Currency)
}

func TestVerifyPayment_TaxAnomalyRoutesToRefundRequired(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	// Tamper booking to have positive tax (tax anomaly scenario)
	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)
	b := f.bookingRepo.bookings[bookingID]
	b.Tax = decimal.NewFromInt(50)
	b.Total = b.Subtotal.Add(b.PlatformFee).Add(b.Tax).Sub(b.Discount)

	// VerifyPayment must return domain.ErrPaymentRefundRequired
	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_tax_anomaly_123",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired), "Must return ErrPaymentRefundRequired")
	assert.Equal(t, "Payment received, but booking could not be completed. Your refund is pending processing.", err.Error())

	// Payment must be marked REFUND_REQUIRED in DB
	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)
	assert.Equal(t, "pay_tax_anomaly_123", *paymentInDB.TransactionID)

	// Booking must be marked CANCELLED in DB
	bookingInDB, err := f.bookingRepo.FindByID(ctx, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, bookingInDB.Status)

	// Reservation must be cancelled
	resInDB, err := f.reservationRepo.FindByID(ctx, f.reservationID)
	require.NoError(t, err)
	assert.Equal(t, domain.ReservationStatusCancelled, resInDB.Status)

	// Finance journal must NOT have been recorded
	assert.Empty(t, f.financeUsecase.capturedInputs)
}

func TestVerifyPayment_TaxAnomalyInventoryFailureDefersToWorker(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)
	b := f.bookingRepo.bookings[bookingID]
	b.Tax = decimal.NewFromInt(50)
	b.Total = b.Subtotal.Add(b.PlatformFee).Add(b.Tax).Sub(b.Discount)

	// Simulate inventory release failure inside savepoint
	f.seatRepo.releaseErr = errors.New("seat lock timeout during release")

	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_tax_inv_fail_123",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	// Outer transaction still committed REFUND_REQUIRED!
	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)

	bookingInDB, err := f.bookingRepo.FindByID(ctx, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, bookingInDB.Status)
}

func TestVerifyPayment_TaxAnomalySavepointControlFailureAborts(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)
	b2 := f.bookingRepo.bookings[bookingID]
	b2.Tax = decimal.NewFromInt(50)
	b2.Total = b2.Subtotal.Add(b2.PlatformFee).Add(b2.Tax).Sub(b2.Discount)

	// Simulate savepoint creation fatal error
	f.txRepos.executeWithSavepointFn = func(name domain.SavepointName, fn func() error) (domain.SavepointResult, error) {
		return domain.SavepointResult{}, domain.ErrSavepointCreateFailed
	}

	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_sp_fail_123",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, domain.ErrSavepointCreateFailed))
}

func TestRazorpayProvider_FindOrderByReceipt_Validation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	provider, err := payment.NewRazorpayProvider("rzp_test_key", "rzp_test_secret", "webhook_secret", logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Empty receipt
	_, err = provider.FindOrderByReceipt(ctx, "", decimal.NewFromInt(100), "INR")
	assert.Error(t, err)

	// Receipt exceeding 40 chars
	_, err = provider.FindOrderByReceipt(ctx, "this_receipt_is_way_too_long_and_exceeds_forty_characters_limit", decimal.NewFromInt(100), "INR")
	assert.Error(t, err)

	// Negative amount
	_, err = provider.FindOrderByReceipt(ctx, "valid_receipt_123", decimal.NewFromInt(-10), "INR")
	assert.Error(t, err)

	// Unsupported currency
	_, err = provider.FindOrderByReceipt(ctx, "valid_receipt_123", decimal.NewFromInt(100), "USD")
	assert.Error(t, err)
}

func TestVerifyPayment_InvoiceLookupFailure_FailsClosed(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	// Simulate unexpected database failure when querying invoice
	dbErr := errors.New("database connection refused")
	f.invoiceRepo.findByBookingErr = dbErr

	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_err_123",
		RazorpaySignature: "sig_dummy",
	})

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "database connection refused")
}

func TestVerifyPayment_ReceiptHealingFailure_FailsClosed(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	// Initial successful verification
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	verifyRes, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_heal_123",
		RazorpaySignature: "sig_dummy",
	})
	require.NoError(t, err)
	require.NotNil(t, verifyRes.Invoice)

	// Simulate missing invoice on subsequent idempotent verification
	f.invoiceRepo.mu.Lock()
	delete(f.invoiceRepo.invoicesByBooking, verifyRes.Booking.ID)
	f.invoiceRepo.mu.Unlock()

	// Inject failure on creation during self-healing
	healErr := errors.New("snapshot creation disk full")
	f.invoiceRepo.createErr = healErr

	// Idempotent retry must fail closed, NOT return success with nil invoice
	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_heal_123",
		RazorpaySignature: "sig_dummy",
	})

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "snapshot creation disk full")
}

func TestVerifyPayment_SuccessReturnsInvoice(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	verifyRes, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_success_123",
		RazorpaySignature: "sig_dummy",
	})
	require.NoError(t, err)
	require.NotNil(t, verifyRes.Invoice)
	assert.Equal(t, verifyRes.Booking.ID, verifyRes.Invoice.BookingID)
	assert.NotEmpty(t, verifyRes.Invoice.InvoiceNumber)
}

func TestPaymentUsecase_RazorpayWebhook_RefundProcessed_Success(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	paymentID := uuid.New()
	bookingID := uuid.New()
	eventID := uuid.New()

	f.paymentRepo.payments[paymentID.String()] = &domain.Payment{
		ID:     paymentID.String(),
		Amount: decimal.NewFromInt(500),
		Status: domain.PaymentStatusSuccess,
	}
	f.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:      bookingID,
		EventID: eventID,
		Status:  domain.BookingStatusConfirmed,
	}

	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		Reason:    "CUSTOMER_CANCELLATION",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_proc_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_test_123",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_proc_001")
	require.NoError(t, err)

	updated, err := f.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, updated.Status)
	assert.NotNil(t, updated.GatewayRefundID)
	assert.Equal(t, "rfnd_test_123", *updated.GatewayRefundID)
	assert.NotNil(t, updated.ProcessedAt)
	require.Len(t, f.financeUsecase.refundInputs, 1)
	assert.Equal(t, refundID, f.financeUsecase.refundInputs[0].RefundID)
}

func TestPaymentUsecase_RazorpayWebhook_RefundProcessed_AlreadySuccess_Idempotent(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	gatewayRefundID := "rfnd_test_123"
	now := time.Now()
	ref := &domain.Refund{
		ID:              refundID,
		BookingID:       uuid.New(),
		PaymentID:       uuid.New(),
		GatewayRefundID: &gatewayRefundID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.RefundStatusSuccess,
		ProcessedAt:     &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_proc_002",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "%s",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, gatewayRefundID, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_proc_002")
	require.NoError(t, err)

	updated, err := f.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, updated.Status)
}

func TestPaymentUsecase_RazorpayWebhook_RefundFailed(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusProcessing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_fail_001",
		"event": "refund.failed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_failed_456",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "failed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_fail_001")
	require.NoError(t, err)

	updated, err := f.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusFailed, updated.Status)
	assert.Equal(t, "rfnd_failed_456", *updated.GatewayRefundID)
}

func TestPaymentUsecase_RazorpayWebhook_RefundFailed_DoesNotOverrideSuccess(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	gatewayRefundID := "rfnd_succ_789"
	now := time.Now()
	ref := &domain.Refund{
		ID:              refundID,
		BookingID:       uuid.New(),
		PaymentID:       uuid.New(),
		GatewayRefundID: &gatewayRefundID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.RefundStatusSuccess,
		ProcessedAt:     &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_fail_late",
		"event": "refund.failed",
		"payload": {
			"refund": {
				"entity": {
					"id": "%s",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "failed",
					"receipt": "%s"
				}
			}
		}
	}`, gatewayRefundID, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_fail_late")
	require.NoError(t, err)

	updated, err := f.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, updated.Status, "Successful refund must NEVER be marked failed")
}

func TestPaymentUsecase_RazorpayWebhook_RefundCreated_Processing(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_created_001",
		"event": "refund.created",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_created_111",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "created",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_created_001")
	require.NoError(t, err)

	updated, err := f.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusProcessing, updated.Status)
	assert.Equal(t, "rfnd_created_111", *updated.GatewayRefundID)
}

func TestPaymentUsecase_RazorpayWebhook_RefundDeduplication(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_dup_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_dup_222",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err1 := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_dup_001")
	require.NoError(t, err1)

	// Second duplicate webhook call with same event ID
	err2 := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_dup_001")
	require.NoError(t, err2, "Duplicate webhook must be acknowledged with nil error")
}

func TestPaymentUsecase_RazorpayWebhook_RefundAmountMismatch(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500), // 500 INR = 50000 paise
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_mismatch_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_mismatch_333",
					"payment_id": "pay_test_xyz",
					"amount": 99900,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_mismatch_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPaymentAmountMismatch))
}

func TestPaymentUsecase_RazorpayWebhook_RefundZeroAmount_Rejected(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_zero_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_zero_333",
					"payment_id": "pay_test_xyz",
					"amount": 0,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_zero_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidPayment))
}

func TestPaymentUsecase_RazorpayWebhook_RefundNegativeAmount_Rejected(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_neg_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_neg_333",
					"payment_id": "pay_test_xyz",
					"amount": -50000,
					"currency": "INR",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_neg_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidPayment))
}

func TestPaymentUsecase_RazorpayWebhook_RefundCurrencyMismatch_Rejected(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, f.refundRepo.Create(ctx, ref))

	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_refund_curr_001",
		"event": "refund.processed",
		"payload": {
			"refund": {
				"entity": {
					"id": "rfnd_curr_333",
					"payment_id": "pay_test_xyz",
					"amount": 50000,
					"currency": "USD",
					"status": "processed",
					"receipt": "%s"
				}
			}
		}
	}`, refundID.String()))

	err := f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_refund_curr_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidPayment))
}

type mockTestRefundProcessor struct {
	mu           sync.Mutex
	processedIDs []uuid.UUID
	processErr   error
	processDelay time.Duration
}

func (m *mockTestRefundProcessor) ProcessPendingRefund(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error) {
	if m.processDelay > 0 {
		time.Sleep(m.processDelay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedIDs = append(m.processedIDs, refundID)
	if m.processErr != nil {
		return nil, m.processErr
	}
	return &domain.Refund{
		ID:     refundID,
		Status: domain.RefundStatusSuccess,
	}, nil
}

func (m *mockTestRefundProcessor) ReconcileStuckRefunds(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	return 0, nil
}

func (m *mockTestRefundProcessor) GetRefundByID(ctx context.Context, refundID uuid.UUID) (*domain.Refund, error) {
	return nil, nil
}

func (m *mockTestRefundProcessor) GetRefundsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	return nil, nil
}

func TestVerifyPayment_CapturedPaymentOnCancelledBooking_RoutesToRefund(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)

	// Mark booking as CANCELLED (e.g. event cancellation happened while payment was in flight)
	b := f.bookingRepo.bookings[bookingID]
	b.Status = domain.BookingStatusCancelled

	proc := &mockTestRefundProcessor{}
	f.usecase.SetRefundProcessor(proc)

	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_cancelled_ev_123",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired), "Must return ErrPaymentRefundRequired")

	// Payment must be marked REFUND_REQUIRED in DB with transaction ID
	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)
	require.NotNil(t, paymentInDB.TransactionID)
	assert.Equal(t, "pay_cancelled_ev_123", *paymentInDB.TransactionID)

	// Booking must remain CANCELLED
	bookingInDB, err := f.bookingRepo.FindByID(ctx, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, bookingInDB.Status)

	// Crucial: No tickets must be issued for cancelled bookings
	assert.Empty(t, f.ticketRepo.tickets, "No tickets should be issued for cancelled bookings")

	// Refund record must be created with correct amount and reason EVENT_CANCELLATION
	paymentUUID, _ := uuid.Parse(p1.ID)
	refunds, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refunds, 1, "Exactly one refund record should be created")
	assert.Equal(t, "EVENT_CANCELLATION", refunds[0].Reason)
	assert.Equal(t, domain.RefundStatusPending, refunds[0].Status)
	assert.True(t, refunds[0].Amount.Equal(p1.Amount), "Refund amount must match payment amount")

	// Financial Ledger: Escrow capture must be recorded
	require.Len(t, f.financeUsecase.escrowInputs, 1, "Escrow capture should be recorded in financial ledger")
	assert.Equal(t, paymentUUID, f.financeUsecase.escrowInputs[0].PaymentID)
	assert.True(t, p1.Amount.Equal(f.financeUsecase.escrowInputs[0].PaymentAmount))
	assert.Equal(t, bookingID, f.financeUsecase.escrowInputs[0].BookingID)
}

func TestVerifyPayment_CancelledEvent_RoutesToRefund(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)

	// Mark event as CANCELLED while booking was still pending
	f.txRepos.eventRepo.status = domain.EventStatusCancelled

	res, err := f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_cancelled_event_456",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)
	assert.Equal(t, "pay_cancelled_event_456", *paymentInDB.TransactionID)

	// Booking transitioned to CANCELLED
	bookingInDB, err := f.bookingRepo.FindByID(ctx, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, bookingInDB.Status)

	// No tickets issued
	assert.Empty(t, f.ticketRepo.tickets)

	// Refund record created
	paymentUUID, _ := uuid.Parse(p1.ID)
	refunds, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refunds, 1)
	assert.Equal(t, "EVENT_CANCELLATION", refunds[0].Reason)
	assert.True(t, refunds[0].Amount.Equal(p1.Amount))
}

func TestProcessRazorpayWebhook_CancelledBooking_CreatesRefund(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)

	// Booking cancelled before webhook arrival
	b := f.bookingRepo.bookings[bookingID]
	b.Status = domain.BookingStatusCancelled

	expectedPaise, _ := domain.ToPaise(p1.Amount)
	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_webhook_cancelled_01",
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "pay_webhook_capt_789",
					"order_id": "%s",
					"amount": %d,
					"currency": "INR",
					"status": "captured"
				}
			}
		}
	}`, *p1.ProviderOrderID, expectedPaise))

	// Webhook must succeed with nil (200 OK to gateway)
	err = f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_webhook_cancelled_01")
	require.NoError(t, err)

	// Payment must be marked REFUND_REQUIRED
	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)
	require.NotNil(t, paymentInDB.TransactionID)
	assert.Equal(t, "pay_webhook_capt_789", *paymentInDB.TransactionID)

	// No tickets issued
	assert.Empty(t, f.ticketRepo.tickets)

	// Refund record created with EVENT_CANCELLATION
	paymentUUID, _ := uuid.Parse(p1.ID)
	refunds, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refunds, 1)
	assert.Equal(t, "EVENT_CANCELLATION", refunds[0].Reason)
	assert.Equal(t, domain.RefundStatusPending, refunds[0].Status)
	assert.True(t, refunds[0].Amount.Equal(p1.Amount))
}

func TestVerifyPayment_And_Webhook_DuplicateHandling_NoDuplicateRefunds(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)

	b := f.bookingRepo.bookings[bookingID]
	b.Status = domain.BookingStatusCancelled

	// 1. VerifyPayment runs first
	_, err = f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_dup_test_111",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	paymentUUID, _ := uuid.Parse(p1.ID)
	refundsAfterVerify, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refundsAfterVerify, 1)

	// 2. Webhook arrives second with same transaction ID
	expectedPaise, _ := domain.ToPaise(p1.Amount)
	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_webhook_dup_02",
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "pay_dup_test_111",
					"order_id": "%s",
					"amount": %d,
					"currency": "INR",
					"status": "captured"
				}
			}
		}
	}`, *p1.ProviderOrderID, expectedPaise))

	err = f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_webhook_dup_02")
	require.NoError(t, err)

	// Still only 1 refund record
	refundsAfterWebhook, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refundsAfterWebhook, 1, "Duplicate webhook must not create second refund")

	// 3. Repeated client VerifyPayment
	_, err = f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_dup_test_111",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	refundsAfterRepeat, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refundsAfterRepeat, 1, "Repeated verification must not create duplicate refund")
}

func TestConcurrent_VerifyPayment_And_Webhook_OnCancelledBooking(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)

	b := f.bookingRepo.bookings[bookingID]
	b.Status = domain.BookingStatusCancelled

	expectedPaise, _ := domain.ToPaise(p1.Amount)
	body := []byte(fmt.Sprintf(`{
		"entity": "event",
		"id": "evt_webhook_conc_03",
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "pay_conc_test_222",
					"order_id": "%s",
					"amount": %d,
					"currency": "INR",
					"status": "captured"
				}
			}
		}
	}`, *p1.ProviderOrderID, expectedPaise))

	var wg sync.WaitGroup
	wg.Add(2)

	var verifyErr error
	var webhookErr error

	go func() {
		defer wg.Done()
		_, verifyErr = f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
			UserID:            f.userID,
			ProviderOrderID:   *p1.ProviderOrderID,
			RazorpayPaymentID: "pay_conc_test_222",
			RazorpaySignature: "sig_dummy",
		})
	}()

	go func() {
		defer wg.Done()
		webhookErr = f.usecase.ProcessRazorpayWebhook(ctx, body, "valid_sig", "evt_webhook_conc_03")
	}()

	wg.Wait()

	require.NoError(t, webhookErr)
	assert.True(t, errors.Is(verifyErr, domain.ErrPaymentRefundRequired))

	paymentUUID, _ := uuid.Parse(p1.ID)
	refunds, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	assert.Len(t, refunds, 1, "Concurrent verify and webhook must create exactly 1 refund record")

	paymentInDB, err := f.paymentRepo.FindByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefundRequired, paymentInDB.Status)
	assert.Empty(t, f.ticketRepo.tickets)
}

func TestGracefulShutdown_TracksBackgroundRefundDispatch(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)
	f.bookingRepo.bookings[bookingID].Status = domain.BookingStatusCancelled

	tracker := worker.NewDispatchTracker()
	proc := &mockTestRefundProcessor{processDelay: 20 * time.Millisecond}

	f.usecase.SetRefundProcessor(proc)
	f.usecase.SetDispatchTracker(tracker)

	_, err = f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_shutdown_track_333",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	// Verify that dispatchTracker properly waited for the background refund task
	done := make(chan struct{})
	go func() {
		tracker.CloseAndWait()
		close(done)
	}()

	select {
	case <-done:
		// Success: background dispatch tracked and waited
	case <-time.After(2 * time.Second):
		t.Fatal("dispatchTracker timed out waiting for background refund dispatch")
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()
	assert.Len(t, proc.processedIDs, 1, "Background refund must have been processed")
}

func TestPaymentUsecase_CancelledBooking_ShutdownActive_DispatchRejected_RefundRemainsPending(t *testing.T) {
	ctx := context.Background()
	f := setupFixture(t)

	// Setup initial pending payment
	p1, err := f.usecase.CreatePayment(ctx, payment.CreatePaymentRequest{
		CheckoutID: f.checkoutID,
		UserID:     f.userID,
		Method:     domain.PaymentMethodCard,
	})
	require.NoError(t, err)

	bookingID, err := uuid.Parse(p1.BookingID)
	require.NoError(t, err)
	f.bookingRepo.bookings[bookingID].Status = domain.BookingStatusCancelled

	tracker := worker.NewDispatchTracker()
	// Shutdown is active; gate is closed
	tracker.Close()

	proc := &mockTestRefundProcessor{}
	f.usecase.SetRefundProcessor(proc)
	f.usecase.SetDispatchTracker(tracker)

	_, err = f.usecase.VerifyPayment(ctx, payment.VerifyPaymentRequest{
		UserID:            f.userID,
		ProviderOrderID:   *p1.ProviderOrderID,
		RazorpayPaymentID: "pay_shutdown_reject_444",
		RazorpaySignature: "sig_dummy",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPaymentRefundRequired))

	// Immediate dispatch must have been rejected
	proc.mu.Lock()
	assert.Empty(t, proc.processedIDs, "Immediate dispatch must be skipped when shutdown is active")
	proc.mu.Unlock()

	// But durable refund must exist with status PENDING in DB
	paymentUUID, err := uuid.Parse(p1.ID)
	require.NoError(t, err)

	refunds, err := f.refundRepo.FindByPaymentID(ctx, paymentUUID)
	require.NoError(t, err)
	require.Len(t, refunds, 1, "Refund record must be created in DB")
	assert.Equal(t, domain.RefundStatusPending, refunds[0].Status)
	assert.Equal(t, "EVENT_CANCELLATION", refunds[0].Reason)
	assert.Equal(t, bookingID, refunds[0].BookingID)
}

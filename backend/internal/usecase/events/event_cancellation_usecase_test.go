package events_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	bookinguc "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	checkoutuc "github.com/adhithyan443/EventHub/backend/internal/usecase/checkout"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	reservationuc "github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Mocks ---

type mockTxManager struct {
	mu    sync.Mutex
	repos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	settlementRepo       *mockSettlementRepo
	scheduleRepo         *mockScheduleRepo
	reservationRepo      *mockReservationRepo
	checkoutRepo         *mockCheckoutRepo
	seatRepo             *mockSeatRepo
	ticketTypeRepo       *mockTicketTypeRepo
	settingRepo          *mockSettingRepo
	pricingTaxRepo       *mockPricingTaxRepo
	userRepo             *mockUserRepo
}

func (m *mockTxRepos) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockTxRepos) OrganizerRepository() domain.OrganizerRepository       { return m.organizerRepo }
func (m *mockTxRepos) BookingRepository() domain.BookingRepository           { return m.bookingRepo }
func (m *mockTxRepos) TicketRepository() domain.TicketRepository             { return m.ticketRepo }
func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository           { return m.paymentRepo }
func (m *mockTxRepos) RefundRepository() domain.RefundRepository             { return m.refundRepo }
func (m *mockTxRepos) SettlementRepository() domain.SettlementRepository     { return m.settlementRepo }
func (m *mockTxRepos) EventScheduleRepository() domain.EventScheduleRepository { return m.scheduleRepo }
func (m *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository { return m.reservationRepo }
func (m *mockTxRepos) CheckoutRepository() domain.CheckoutRepository         { return m.checkoutRepo }
func (m *mockTxRepos) SeatRepository() domain.SeatRepository                 { return m.seatRepo }
func (m *mockTxRepos) TicketTypeRepository() domain.TicketTypeRepository     { return m.ticketTypeRepo }
func (m *mockTxRepos) EventSettingRepository() domain.EventSettingRepository { return m.settingRepo }
func (m *mockTxRepos) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return m.pricingTaxRepo
}
func (m *mockTxRepos) UserRepository() domain.UserRepository {
	if m == nil || m.userRepo == nil {
		return nil
	}
	return m.userRepo
}

type mockUserRepo struct {
	domain.UserRepository
	mu      sync.Mutex
	users   map[uuid.UUID]*domain.User
	findErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	copied := *u
	return &copied, nil
}

type mockCancellationNotifier struct {
	mu               sync.Mutex
	customerNotices  []domain.CustomerCancellationNotice
	organizerNotices []domain.OrganizerCancellationNotice
	customerErr      error
	organizerErr     error
	notifyCustCh     chan domain.CustomerCancellationNotice
	notifyOrgCh      chan domain.OrganizerCancellationNotice
}

func newMockCancellationNotifier() *mockCancellationNotifier {
	return &mockCancellationNotifier{
		notifyCustCh: make(chan domain.CustomerCancellationNotice, 50),
		notifyOrgCh:  make(chan domain.OrganizerCancellationNotice, 50),
	}
}

func (m *mockCancellationNotifier) NotifyCustomer(ctx context.Context, notice domain.CustomerCancellationNotice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customerNotices = append(m.customerNotices, notice)
	select {
	case m.notifyCustCh <- notice:
	default:
	}
	return m.customerErr
}

func (m *mockCancellationNotifier) NotifyOrganizer(ctx context.Context, notice domain.OrganizerCancellationNotice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.organizerNotices = append(m.organizerNotices, notice)
	select {
	case m.notifyOrgCh <- notice:
	default:
	}
	return m.organizerErr
}

type mockSettlementRepo struct {
	domain.SettlementRepository
	mu          sync.Mutex
	settlements map[uuid.UUID]*domain.Settlement
	updateErr   error
}

func newMockSettlementRepo() *mockSettlementRepo {
	return &mockSettlementRepo{settlements: make(map[uuid.UUID]*domain.Settlement)}
}

func (m *mockSettlementRepo) List(ctx context.Context, filter domain.SettlementFilter) (*domain.SettlementListResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var matched []*domain.Settlement
	for _, s := range m.settlements {
		if filter.EventID != nil && (s.EventID == nil || *s.EventID != *filter.EventID) {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		copied := *s
		matched = append(matched, &copied)
	}
	return &domain.SettlementListResult{
		Settlements: matched,
		Total:       int64(len(matched)),
	}, nil
}

func (m *mockSettlementRepo) Update(ctx context.Context, s *domain.Settlement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	copied := *s
	m.settlements[s.ID] = &copied
	return nil
}

type mockScheduleRepo struct {
	domain.EventScheduleRepository
	mu        sync.Mutex
	schedules map[uuid.UUID]*domain.EventSchedule
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{schedules: make(map[uuid.UUID]*domain.EventSchedule)}
}

func (m *mockScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.schedules[eventID]; ok {
		copied := *s
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}

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

func (m *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	org, ok := m.organizers[id]
	if !ok {
		return nil, domain.ErrOrganizerNotFound
	}
	copied := *org
	return &copied, nil
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

func (m *mockBookingRepo) Create(ctx context.Context, b *domain.Booking, items []domain.BookingItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *b
	m.bookings[b.ID] = &copied
	return nil
}

func (m *mockBookingRepo) CreateItem(ctx context.Context, item *domain.BookingItem) error {
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

type mockReservationRepo struct {
	domain.SeatReservationRepository
	mu           sync.Mutex
	reservations map[uuid.UUID]*domain.SeatReservation
	items        map[uuid.UUID][]domain.SeatReservationItem
}

func newMockReservationRepo() *mockReservationRepo {
	return &mockReservationRepo{
		reservations: make(map[uuid.UUID]*domain.SeatReservation),
		items:        make(map[uuid.UUID][]domain.SeatReservationItem),
	}
}

func (m *mockReservationRepo) FindActiveByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.SeatReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.SeatReservation
	for _, r := range m.reservations {
		if r.EventID == eventID && r.Status == domain.ReservationStatusActive {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		copied := *r
		return &copied, nil
	}
	return nil, domain.ErrSeatReservationNotFound
}

func (m *mockReservationRepo) FindItemsByReservationID(ctx context.Context, id uuid.UUID) ([]domain.SeatReservationItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[id], nil
}

func (m *mockReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		r.Status = status
		return nil
	}
	return domain.ErrSeatReservationNotFound
}

func (m *mockReservationRepo) Create(ctx context.Context, r *domain.SeatReservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *r
	m.reservations[r.ID] = &copied
	return nil
}

func (m *mockReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ReservationID] = append(m.items[item.ReservationID], *item)
	return nil
}

type mockCheckoutRepo struct {
	domain.CheckoutRepository
	mu        sync.Mutex
	checkouts map[uuid.UUID]*domain.Checkout
}

func newMockCheckoutRepo() *mockCheckoutRepo {
	return &mockCheckoutRepo{
		checkouts: make(map[uuid.UUID]*domain.Checkout),
	}
}

func (m *mockCheckoutRepo) Create(ctx context.Context, c *domain.Checkout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *c
	m.checkouts[c.ID] = &copied
	return nil
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

func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, resID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	updated := false
	for _, c := range m.checkouts {
		if c.ReservationID == resID && c.Status == domain.CheckoutStatusActive {
			c.Status = domain.CheckoutStatusExpired
			updated = true
		}
	}
	return updated, nil
}

type mockSeatRepo struct {
	domain.SeatRepository
	mu            sync.Mutex
	seats         map[uuid.UUID]*domain.Seat
	releasedSeats []uuid.UUID
}

func newMockSeatRepo() *mockSeatRepo {
	return &mockSeatRepo{
		seats: make(map[uuid.UUID]*domain.Seat),
	}
}

func (m *mockSeatRepo) ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.releasedSeats = append(m.releasedSeats, seatID)
	if s, ok := m.seats[seatID]; ok {
		s.Status = domain.SeatStatusAvailable
	}
	return nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	mu          sync.Mutex
	ticketTypes map[uuid.UUID]*domain.TicketType
	released    map[uuid.UUID]int
}

func newMockTicketTypeRepo() *mockTicketTypeRepo {
	return &mockTicketTypeRepo{
		ticketTypes: make(map[uuid.UUID]*domain.TicketType),
		released:    make(map[uuid.UUID]int),
	}
}

func (m *mockTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, ttID uuid.UUID, qty int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.released[ttID] += qty
	if tt, ok := m.ticketTypes[ttID]; ok {
		tt.AvailableQuantity += qty
	}
	return nil
}

func (m *mockTicketTypeRepo) ReserveQuantityIfAvailable(ctx context.Context, ttID uuid.UUID, qty int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tt, ok := m.ticketTypes[ttID]
	if !ok {
		return domain.ErrTicketTypeNotFound
	}
	if tt.AvailableQuantity < qty {
		return domain.ErrInsufficientTicketQuantity
	}
	tt.AvailableQuantity -= qty
	return nil
}

func (m *mockTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.TicketType
	for _, tt := range m.ticketTypes {
		if tt.EventID == eventID {
			copied := *tt
			result = append(result, &copied)
		}
	}
	return result, nil
}

type mockSettingRepo struct {
	domain.EventSettingRepository
	mu       sync.Mutex
	settings map[uuid.UUID]*domain.EventSetting
}

func newMockSettingRepo() *mockSettingRepo {
	return &mockSettingRepo{settings: make(map[uuid.UUID]*domain.EventSetting)}
}

func (m *mockSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.settings[eventID]; ok {
		copied := *s
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}

type mockPricingTaxRepo struct {
	domain.PricingTaxConfigurationRepository
}

func (m *mockPricingTaxRepo) FindActiveAt(ctx context.Context, t time.Time) (*domain.PricingTaxConfiguration, error) {
	return &domain.PricingTaxConfiguration{
		PlatformFeeEnabled: false,
		TaxEnabled:         false,
	}, nil
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
	userRepo        *mockUserRepo
	notifier        *mockCancellationNotifier
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
	settlementRepo := newMockSettlementRepo()
	scheduleRepo := newMockScheduleRepo()
	reservationRepo := newMockReservationRepo()
	checkoutRepo := newMockCheckoutRepo()
	seatRepo := newMockSeatRepo()
	ticketTypeRepo := newMockTicketTypeRepo()
	settingRepo := newMockSettingRepo()
	pricingTaxRepo := &mockPricingTaxRepo{}
	userRepo := newMockUserRepo()

	txRepos := &mockTxRepos{
		eventRepo:        eventRepo,
		cancellationRepo: cancellationRepo,
		organizerRepo:    organizerRepo,
		bookingRepo:      bookingRepo,
		ticketRepo:       ticketRepo,
		paymentRepo:      paymentRepo,
		refundRepo:       refundRepo,
		settlementRepo:   settlementRepo,
		scheduleRepo:     scheduleRepo,
		reservationRepo:  reservationRepo,
		checkoutRepo:     checkoutRepo,
		seatRepo:         seatRepo,
		ticketTypeRepo:   ticketTypeRepo,
		settingRepo:      settingRepo,
		pricingTaxRepo:   pricingTaxRepo,
		userRepo:         userRepo,
	}

	txMgr := &mockTxManager{repos: txRepos}
	proc := &mockRefundProcessor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := events.NewEventCancellationUsecase(txMgr, proc, logger)
	notifier := newMockCancellationNotifier()
	uc.SetNotifier(notifier)

	// Setup active organizer user
	userRepo.users[orgUserID] = &domain.User{
		ID:       orgUserID,
		FullName: "Test Organizer",
		Email:    "organizer@example.com",
		Role:     "ORGANIZER",
	}

	// Setup active organizer
	organizer := &domain.Organizer{
		ID:     orgID,
		UserID: orgUserID,
		Status: "ACTIVE",
	}
	organizerRepo.organizers[orgID] = organizer

	// Setup published event with future schedule
	now := time.Now().UTC()
	event := &domain.Event{
		ID:          evID,
		OrganizerID: orgID,
		Title:       "Tech World Summit 2026",
		Status:      domain.EventStatusPublished,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now.Add(-24 * time.Hour),
	}
	eventRepo.events[evID] = event

	schedule := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   evID,
		EventDate: now.Add(48 * time.Hour),
		StartTime: now.Add(48 * time.Hour),
		EndTime:   now.Add(52 * time.Hour),
		IsAllDay:  false,
	}
	scheduleRepo.schedules[evID] = schedule

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
		userRepo:        userRepo,
		notifier:        notifier,
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

func TestEventCancellationUsecase_CancelEvent_AlreadyCancelled_ReturnsError(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// 1st cancel call succeeds
	sum1, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Initial cancellation")
	require.NoError(t, err)
	assert.Equal(t, domain.EventStatusCancelled, sum1.EventStatus)

	// 2nd cancel call returns ErrEventAlreadyCancelled
	sum2, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Repeated cancel call")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrEventAlreadyCancelled))
	assert.Nil(t, sum2)
}

func TestEventCancellationUsecase_CancelEvent_DraftEvent_Rejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Set event status to DRAFT
	f.event.Status = domain.EventStatusDraft

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Try to cancel draft")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDraftEventCannotBeCancelled))
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelled))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusDraft, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_PastConcludedEvent_Rejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Set schedule in the past
	now := time.Now().UTC()
	f.txRepos.scheduleRepo.schedules[f.eventID] = &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   f.eventID,
		EventDate: now.Add(-72 * time.Hour),
		StartTime: now.Add(-72 * time.Hour),
		EndTime:   now.Add(-68 * time.Hour),
		IsAllDay:  false,
	}

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Try to cancel past event")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPastEventCannotBeCancelled))
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelled))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_MissingReason_Rejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Empty reason
	sum1, err1 := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "")
	require.Error(t, err1)
	assert.True(t, errors.Is(err1, domain.ErrCancellationReasonRequired))
	assert.Nil(t, sum1)

	// Whitespace-only reason
	sum2, err2 := f.usecase.CancelEventByAdmin(ctx, uuid.New(), f.eventID, "    ")
	require.Error(t, err2)
	assert.True(t, errors.Is(err2, domain.ErrCancellationReasonRequired))
	assert.Nil(t, sum2)

	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_ReasonTooLong_Rejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	tooLong := strings.Repeat("a", 1001)
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, tooLong)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCancellationReasonTooLong))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_CompletedSettlement_BlocksCancellation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Add completed settlement for this event
	settlementID := uuid.New()
	completedSettlement := &domain.Settlement{
		ID:          settlementID,
		OrganizerID: f.organizerID,
		EventID:     &f.eventID,
		Amount:      decimal.NewFromInt(5000),
		Currency:    "INR",
		Status:      domain.SettlementStatusCompleted,
		CreatedAt:   time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().UTC().Add(-24 * time.Hour),
	}
	f.txRepos.settlementRepo.settlements[settlementID] = completedSettlement

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Venue issue")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelledWithCompletedSettlement))
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelled))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_ApprovedSettlement_BlocksCancellation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Add approved settlement
	settlementID := uuid.New()
	approvedSettlement := &domain.Settlement{
		ID:          settlementID,
		OrganizerID: f.organizerID,
		EventID:     &f.eventID,
		Amount:      decimal.NewFromInt(3000),
		Currency:    "INR",
		Status:      domain.SettlementStatusApproved,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	f.txRepos.settlementRepo.settlements[settlementID] = approvedSettlement

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Venue issue")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelledWithActiveSettlement))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_ProcessingSettlement_BlocksCancellation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Add processing settlement
	settlementID := uuid.New()
	procSettlement := &domain.Settlement{
		ID:          settlementID,
		OrganizerID: f.organizerID,
		EventID:     &f.eventID,
		Amount:      decimal.NewFromInt(3000),
		Currency:    "INR",
		Status:      domain.SettlementStatusProcessing,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	f.txRepos.settlementRepo.settlements[settlementID] = procSettlement

	summary, err := f.usecase.CancelEventByAdmin(ctx, uuid.New(), f.eventID, "Admin cancellation")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrEventCannotBeCancelledWithActiveSettlement))
	assert.Nil(t, summary)
	assert.Equal(t, domain.EventStatusPublished, f.event.Status)
}

func TestEventCancellationUsecase_CancelEvent_PendingSettlement_SafelyRejected(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	// Add pending settlement for this event
	settlementID := uuid.New()
	pendingSettlement := &domain.Settlement{
		ID:          settlementID,
		OrganizerID: f.organizerID,
		EventID:     &f.eventID,
		Amount:      decimal.NewFromInt(2500),
		Currency:    "INR",
		Status:      domain.SettlementStatusPending,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}
	f.txRepos.settlementRepo.settlements[settlementID] = pendingSettlement

	reason := "Emergency government curfew imposed"
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, reason)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, domain.EventStatusCancelled, summary.EventStatus)

	// Verify pending settlement was safely rejected with audit history
	updatedSettlement := f.txRepos.settlementRepo.settlements[settlementID]
	require.NotNil(t, updatedSettlement)
	assert.Equal(t, domain.SettlementStatusRejected, updatedSettlement.Status)
	require.NotNil(t, updatedSettlement.RejectedAt)
	require.NotNil(t, updatedSettlement.RejectionReason)
	assert.Contains(t, *updatedSettlement.RejectionReason, "Automatically rejected due to event cancellation")
	assert.Contains(t, *updatedSettlement.RejectionReason, reason)
}

func TestEventCancellationUsecase_CancelEvent_ConcurrentRequests_PreventDuplicates(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := f.usecase.CancelEventByOrganizer(
				ctx,
				f.organizerUserID,
				f.eventID,
				fmt.Sprintf("Concurrent cancellation attempt %d", idx),
			)
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	alreadyCancelledCount := 0
	for err := range results {
		if err == nil {
			successCount++
		} else if errors.Is(err, domain.ErrEventAlreadyCancelled) {
			alreadyCancelledCount++
		}
	}

	// Exactly one request must succeed, all other serialized concurrent attempts must be rejected with ErrEventAlreadyCancelled
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 4, alreadyCancelledCount)
	assert.Equal(t, domain.EventStatusCancelled, f.txRepos.eventRepo.events[f.eventID].Status)
}

func TestEventCancellationUsecase_CancelEvent_TransactionRollback_OnFailure(t *testing.T) {
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

	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Curfew declared")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated database failure")
	assert.Nil(t, summary)
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

func TestEventCancellationUsecase_ActiveReservationsAndInventoryCleanedUp(t *testing.T) {
	ctx := context.Background()
	f := newCancellationFixture(t)

	// 1. Setup active seat reservation
	seatID := uuid.New()
	f.txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusReserved,
	}

	seatResID := uuid.New()
	seatUserID := uuid.New()
	f.txRepos.reservationRepo.reservations[seatResID] = &domain.SeatReservation{
		ID:        seatResID,
		UserID:    seatUserID,
		EventID:   f.eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	f.txRepos.reservationRepo.items[seatResID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: seatResID,
			SeatID:        &seatID,
			Quantity:      1,
			Price:         decimal.NewFromInt(150),
		},
	}

	seatCheckoutID := uuid.New()
	f.txRepos.checkoutRepo.checkouts[seatCheckoutID] = &domain.Checkout{
		ID:            seatCheckoutID,
		ReservationID: seatResID,
		UserID:        seatUserID,
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     time.Now().Add(15 * time.Minute),
	}

	// 2. Setup active GA reservation
	ttID := uuid.New()
	f.txRepos.ticketTypeRepo.ticketTypes[ttID] = &domain.TicketType{
		ID:                ttID,
		EventID:           f.eventID,
		TotalQuantity:     50,
		AvailableQuantity: 10,
	}

	gaResID := uuid.New()
	gaUserID := uuid.New()
	f.txRepos.reservationRepo.reservations[gaResID] = &domain.SeatReservation{
		ID:        gaResID,
		UserID:    gaUserID,
		EventID:   f.eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	f.txRepos.reservationRepo.items[gaResID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: gaResID,
			TicketTypeID:  &ttID,
			Quantity:      3,
			Price:         decimal.NewFromInt(50),
		},
	}

	gaCheckoutID := uuid.New()
	f.txRepos.checkoutRepo.checkouts[gaCheckoutID] = &domain.Checkout{
		ID:            gaCheckoutID,
		ReservationID: gaResID,
		UserID:        gaUserID,
		Status:        domain.CheckoutStatusActive,
		ExpiresAt:     time.Now().Add(10 * time.Minute),
	}

	// 3. Setup an already cancelled reservation to ensure it is not touched
	alreadyCancelledResID := uuid.New()
	f.txRepos.reservationRepo.reservations[alreadyCancelledResID] = &domain.SeatReservation{
		ID:        alreadyCancelledResID,
		UserID:    uuid.New(),
		EventID:   f.eventID,
		Status:    domain.ReservationStatusCancelled,
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}

	// Execute event cancellation
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Venue double-booked")
	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Verify event status
	f.txRepos.eventRepo.mu.Lock()
	assert.Equal(t, domain.EventStatusCancelled, f.txRepos.eventRepo.events[f.eventID].Status)
	f.txRepos.eventRepo.mu.Unlock()

	// Verify reservations transitioned to CANCELLED
	f.txRepos.reservationRepo.mu.Lock()
	assert.Equal(t, domain.ReservationStatusCancelled, f.txRepos.reservationRepo.reservations[seatResID].Status)
	assert.Equal(t, domain.ReservationStatusCancelled, f.txRepos.reservationRepo.reservations[gaResID].Status)
	assert.Equal(t, domain.ReservationStatusCancelled, f.txRepos.reservationRepo.reservations[alreadyCancelledResID].Status)
	f.txRepos.reservationRepo.mu.Unlock()

	// Verify active checkouts transitioned to EXPIRED
	f.txRepos.checkoutRepo.mu.Lock()
	assert.Equal(t, domain.CheckoutStatusExpired, f.txRepos.checkoutRepo.checkouts[seatCheckoutID].Status)
	assert.Equal(t, domain.CheckoutStatusExpired, f.txRepos.checkoutRepo.checkouts[gaCheckoutID].Status)
	f.txRepos.checkoutRepo.mu.Unlock()

	// Verify reserved seat was released to AVAILABLE
	f.txRepos.seatRepo.mu.Lock()
	assert.Equal(t, domain.SeatStatusAvailable, f.txRepos.seatRepo.seats[seatID].Status)
	assert.Contains(t, f.txRepos.seatRepo.releasedSeats, seatID)
	f.txRepos.seatRepo.mu.Unlock()

	// Verify GA ticket quantity was released (10 + 3 = 13)
	f.txRepos.ticketTypeRepo.mu.Lock()
	assert.Equal(t, 13, f.txRepos.ticketTypeRepo.ticketTypes[ttID].AvailableQuantity)
	assert.Equal(t, 3, f.txRepos.ticketTypeRepo.released[ttID])
	f.txRepos.ticketTypeRepo.mu.Unlock()
}

func TestEventCancellationUsecase_BookedSeatsAndConfirmedInventoryPreserved(t *testing.T) {
	ctx := context.Background()
	f := newCancellationFixture(t)

	// Booked seat
	seatID := uuid.New()
	f.txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusBooked,
	}

	// Ticket type with sold/booked tickets (total 50, available 40 -> 10 sold)
	ttID := uuid.New()
	f.txRepos.ticketTypeRepo.ticketTypes[ttID] = &domain.TicketType{
		ID:                ttID,
		EventID:           f.eventID,
		TotalQuantity:     50,
		AvailableQuantity: 40,
	}

	// Confirmed booking with 1 valid ticket and booked seat
	bID := uuid.New()
	tID := uuid.New()
	pID := uuid.New()

	f.txRepos.bookingRepo.bookings[bID] = &domain.Booking{
		ID:          bID,
		EventID:     f.eventID,
		Status:      domain.BookingStatusConfirmed,
		Total:       decimal.NewFromInt(100),
		Subtotal:    decimal.NewFromInt(100),
		PlatformFee: decimal.Zero,
		Tax:         decimal.Zero,
	}
	f.txRepos.ticketRepo.tickets[tID] = &domain.Ticket{
		ID:        tID,
		BookingID: bID,
		EventID:   f.eventID,
		Price:     decimal.NewFromInt(100),
		Status:    domain.TicketStatusValid,
	}

	gatewayTxID := "pay_confirmed_preserve_111"
	f.txRepos.paymentRepo.payments[pID.String()] = &domain.Payment{
		ID:            pID.String(),
		BookingID:     bID.String(),
		Amount:        decimal.NewFromInt(100),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &gatewayTxID,
	}

	// Execute event cancellation
	summary, err := f.usecase.CancelEventByOrganizer(ctx, f.organizerUserID, f.eventID, "Extreme weather forecast")
	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Booking and ticket must be cancelled
	f.txRepos.bookingRepo.mu.Lock()
	assert.Equal(t, domain.BookingStatusCancelled, f.txRepos.bookingRepo.bookings[bID].Status)
	f.txRepos.bookingRepo.mu.Unlock()

	f.txRepos.ticketRepo.mu.Lock()
	assert.Equal(t, domain.TicketStatusCancelled, f.txRepos.ticketRepo.tickets[tID].Status)
	f.txRepos.ticketRepo.mu.Unlock()

	// CRITICAL: Booked seat must NOT be released back to AVAILABLE. It must remain BOOKED to preserve audit history!
	f.txRepos.seatRepo.mu.Lock()
	assert.Equal(t, domain.SeatStatusBooked, f.txRepos.seatRepo.seats[seatID].Status)
	assert.NotContains(t, f.txRepos.seatRepo.releasedSeats, seatID)
	f.txRepos.seatRepo.mu.Unlock()

	// CRITICAL: Ticket type available quantity must NOT be incremented for confirmed bookings (retaining tickets_sold analytics)
	f.txRepos.ticketTypeRepo.mu.Lock()
	assert.Equal(t, 40, f.txRepos.ticketTypeRepo.ticketTypes[ttID].AvailableQuantity)
	assert.Equal(t, 0, f.txRepos.ticketTypeRepo.released[ttID])
	f.txRepos.ticketTypeRepo.mu.Unlock()
}

func TestEventCancellation_Concurrency_RacingWithReservationCreation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	resLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resUC := reservationuc.NewReservationUsecase(f.txManager, resLogger)

	ttID := uuid.New()
	f.txRepos.ticketTypeRepo.ticketTypes[ttID] = &domain.TicketType{
		ID:                ttID,
		EventID:           f.eventID,
		TotalQuantity:     100,
		AvailableQuantity: 100,
	}

	setting := &domain.EventSetting{
		EventID:             f.eventID,
		BookingLimitPerUser: 10,
	}
	f.txRepos.settingRepo.settings[f.eventID] = setting

	const concurrentWorkers = 8
	var wg sync.WaitGroup
	resErrors := make([]error, concurrentWorkers)
	resResults := make([]*domain.SeatReservation, concurrentWorkers)

	startBarrier := make(chan struct{})

	for i := 0; i < concurrentWorkers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier
			req := reservationuc.CreateReservationRequest{
				UserID:  uuid.New(),
				EventID: f.eventID,
				Tickets: []reservationuc.TicketSelection{
					{TicketTypeID: ttID, Quantity: 2},
				},
			}
			res, err := resUC.CreateReservation(ctx, req)
			resResults[idx] = res
			resErrors[idx] = err
		}(i)
	}

	var cancelErr error
	var cancelSummary *domain.EventCancellationSummary
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startBarrier
		cancelSummary, cancelErr = f.usecase.CancelEventByOrganizer(
			ctx,
			f.organizerUserID,
			f.eventID,
			"Concurrent cancellation with reservation",
		)
	}()

	close(startBarrier)
	wg.Wait()

	require.NoError(t, cancelErr)
	assert.NotNil(t, cancelSummary)

	// Invariants:
	// 1. Any reservation serialized after cancellation must return ErrEventNotPublic
	// 2. Any reservation committed before cancellation must be transitioned to CANCELLED
	// 3. Exactly zero active reservations must remain for this event
	activeCount := 0
	f.txRepos.reservationRepo.mu.Lock()
	for _, r := range f.txRepos.reservationRepo.reservations {
		if r.EventID == f.eventID && r.Status == domain.ReservationStatusActive {
			activeCount++
		}
	}
	f.txRepos.reservationRepo.mu.Unlock()
	assert.Equal(t, 0, activeCount, "No active reservation can remain after event cancellation")

	// 4. Ticket inventory must be completely restored to 100
	f.txRepos.ticketTypeRepo.mu.Lock()
	assert.Equal(t, 100, f.txRepos.ticketTypeRepo.ticketTypes[ttID].AvailableQuantity, "Ticket quantity must be completely restored")
	f.txRepos.ticketTypeRepo.mu.Unlock()
}

func TestEventCancellation_Concurrency_RacingWithCheckoutCreation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	checkoutLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	checkoutUC := checkoutuc.NewCheckoutUsecase(f.txManager, checkoutLogger)

	// Setup active reservation
	resID := uuid.New()
	userID := uuid.New()
	f.txRepos.reservationRepo.reservations[resID] = &domain.SeatReservation{
		ID:        resID,
		UserID:    userID,
		EventID:   f.eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	f.txRepos.reservationRepo.items[resID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: resID,
			Quantity:      1,
			Price:         decimal.NewFromInt(100),
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	startBarrier := make(chan struct{})

	var checkoutErr error
	var chk *domain.Checkout
	go func() {
		defer wg.Done()
		<-startBarrier
		chk, checkoutErr = checkoutUC.CreateCheckout(ctx, checkoutuc.CreateCheckoutRequest{
			ReservationID: resID,
			UserID:        userID,
		})
	}()

	var cancelErr error
	go func() {
		defer wg.Done()
		<-startBarrier
		_, cancelErr = f.usecase.CancelEventByOrganizer(
			ctx,
			f.organizerUserID,
			f.eventID,
			"Cancellation racing with checkout",
		)
	}()

	close(startBarrier)
	wg.Wait()

	require.NoError(t, cancelErr)

	// Invariants:
	// 1. Reservation must be CANCELLED
	f.txRepos.reservationRepo.mu.Lock()
	assert.Equal(t, domain.ReservationStatusCancelled, f.txRepos.reservationRepo.reservations[resID].Status)
	f.txRepos.reservationRepo.mu.Unlock()

	// 2. In NO case can any checkout remain in ACTIVE status
	f.txRepos.checkoutRepo.mu.Lock()
	for _, c := range f.txRepos.checkoutRepo.checkouts {
		if c.ReservationID == resID {
			assert.Equal(t, domain.CheckoutStatusExpired, c.Status, "Checkout must be expired after cancellation")
		}
	}
	f.txRepos.checkoutRepo.mu.Unlock()

	if checkoutErr != nil {
		assert.True(t, errors.Is(checkoutErr, domain.ErrEventNotPublic) || errors.Is(checkoutErr, domain.ErrInvalidReservationRequest))
	} else {
		require.NotNil(t, chk)
		f.txRepos.checkoutRepo.mu.Lock()
		dbChk := f.txRepos.checkoutRepo.checkouts[chk.ID]
		f.txRepos.checkoutRepo.mu.Unlock()
		assert.Equal(t, domain.CheckoutStatusExpired, dbChk.Status)
	}
}

func TestEventCancellation_Concurrency_RacingWithBookingCreation(t *testing.T) {
	f := newCancellationFixture(t)
	ctx := context.Background()

	bookingLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bookingUC := bookinguc.NewBookingUsecase(f.txManager, nil, bookingLogger)

	// Setup active reservation and active checkout
	resID := uuid.New()
	userID := uuid.New()
	checkoutID := uuid.New()
	now := time.Now()

	f.txRepos.reservationRepo.reservations[resID] = &domain.SeatReservation{
		ID:        resID,
		UserID:    userID,
		EventID:   f.eventID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(15 * time.Minute),
	}
	f.txRepos.reservationRepo.items[resID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: resID,
			Quantity:      1,
			Price:         decimal.NewFromInt(100),
		},
	}

	f.txRepos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:             checkoutID,
		ReservationID:  resID,
		UserID:         userID,
		Subtotal:       decimal.NewFromInt(100),
		PlatformFee:    decimal.Zero,
		TaxAmount:      decimal.Zero,
		DiscountAmount: decimal.Zero,
		TotalAmount:    decimal.NewFromInt(100),
		Status:         domain.CheckoutStatusActive,
		ExpiresAt:      now.Add(15 * time.Minute),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	startBarrier := make(chan struct{})

	var bookingErr error
	var bkg *domain.Booking
	go func() {
		defer wg.Done()
		<-startBarrier
		bkg, bookingErr = bookingUC.CreateBooking(ctx, bookinguc.CreateBookingRequest{
			CheckoutID: checkoutID,
			UserID:     userID,
		})
	}()

	var cancelErr error
	go func() {
		defer wg.Done()
		<-startBarrier
		_, cancelErr = f.usecase.CancelEventByOrganizer(
			ctx,
			f.organizerUserID,
			f.eventID,
			"Cancellation racing with booking",
		)
	}()

	close(startBarrier)
	wg.Wait()

	require.NoError(t, cancelErr)

	// Invariants:
	// 1. Any booking created before cancellation must be CANCELLED by event cancellation
	// 2. If cancellation committed first, CreateBooking must return ErrEventNotPublic or ErrInvalidReservationRequest
	// 3. ZERO bookings in PENDING or CONFIRMED status can remain
	f.txRepos.bookingRepo.mu.Lock()
	for _, b := range f.txRepos.bookingRepo.bookings {
		if b.EventID == f.eventID {
			assert.Equal(t, domain.BookingStatusCancelled, b.Status, "All bookings for cancelled event must be CANCELLED")
		}
	}
	f.txRepos.bookingRepo.mu.Unlock()

	if bookingErr != nil {
		assert.True(t, errors.Is(bookingErr, domain.ErrEventNotPublic) || errors.Is(bookingErr, domain.ErrInvalidReservationRequest))
	} else {
		require.NotNil(t, bkg)
		f.txRepos.bookingRepo.mu.Lock()
		dbBkg := f.txRepos.bookingRepo.bookings[bkg.ID]
		f.txRepos.bookingRepo.mu.Unlock()
		assert.Equal(t, domain.BookingStatusCancelled, dbBkg.Status)
	}
}

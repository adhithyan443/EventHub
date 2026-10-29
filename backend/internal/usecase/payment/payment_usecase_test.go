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
	bookinguc "github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations ---

type mockPaymentRepo struct {
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
	return true, nil
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
func (m *mockTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}

type mockSeatRepo struct {
	domain.SeatRepository
}

func (m *mockSeatRepo) BookReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	return nil
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
	paymentRepo           *mockPaymentRepo
	bookingRepo           *mockBookingRepo
	checkoutRepo          *mockCheckoutRepo
	seatReservationRepo   *mockSeatReservationRepo
	ticketRepo            *mockTicketRepo
	seatRepo              *mockSeatRepo
	webhookEventRepo      *mockRazorpayWebhookEventRepo
	eventCancellationRepo *mockEventCancellationRepo
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
	return &mockRefundRepo{}
}

type mockRefundRepo struct {
	domain.RefundRepository
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
	webhookEventRepo := &mockRazorpayWebhookEventRepo{events: make(map[string]string)}

	txRepos := &mockTxRepos{
		paymentRepo:           paymentRepo,
		bookingRepo:           bookingRepo,
		checkoutRepo:          checkoutRepo,
		seatReservationRepo:   reservationRepo,
		ticketRepo:            ticketRepo,
		seatRepo:              seatRepo,
		webhookEventRepo:      webhookEventRepo,
		eventCancellationRepo: &mockEventCancellationRepo{cancellations: make(map[uuid.UUID]*domain.EventCancellation)},
	}
	tm := &mockTransactionManager{repos: txRepos}

	provider := newMockPaymentProvider()
	bookingUC := bookinguc.NewBookingUsecase(tm, provider, logger)
	paymentUC := payment.NewPaymentUsecase(tm, bookingUC, provider, logger)

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

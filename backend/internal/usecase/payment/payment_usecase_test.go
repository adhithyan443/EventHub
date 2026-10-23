package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	paymentuc "github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPaymentProvider struct {
	verifyWebhookSignatureFunc func(body []byte, signature string) bool
}

func (m *mockPaymentProvider) Initiate(ctx context.Context, amount decimal.Decimal, currency string, idempotencyKey string) (string, error) {
	return "order_test_123", nil
}

func (m *mockPaymentProvider) Verify(ctx context.Context, providerOrderID, paymentID, signature string, expectedAmount decimal.Decimal, currency string) (bool, error) {
	return true, nil
}

func (m *mockPaymentProvider) VerifyWebhookSignature(body []byte, signature string) bool {
	if m.verifyWebhookSignatureFunc != nil {
		return m.verifyWebhookSignatureFunc(body, signature)
	}
	return true
}

type mockWebhookEventRepo struct {
	createFunc func(ctx context.Context, eventID, eventType string) error
	created    []string
}

func (m *mockWebhookEventRepo) Create(ctx context.Context, eventID, eventType string) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, eventID, eventType)
	}
	m.created = append(m.created, eventID)
	return nil
}

type mockPaymentRepo struct {
	payments           map[string]*domain.Payment
	markSuccessfulFunc func(ctx context.Context, paymentID string, transactionID string, paidAt time.Time) error
	markFailedFunc     func(ctx context.Context, paymentID string) error
}

func (m *mockPaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *mockPaymentRepo) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	var list []domain.Payment
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockPaymentRepo) Update(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *mockPaymentRepo) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Payment, error) {
	for _, p := range m.payments {
		if p.ProviderOrderID != nil && *p.ProviderOrderID == providerOrderID {
			return p, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) MarkSuccessful(ctx context.Context, paymentID string, transactionID string, paidAt time.Time) error {
	if m.markSuccessfulFunc != nil {
		return m.markSuccessfulFunc(ctx, paymentID, transactionID, paidAt)
	}
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if p.Status != domain.PaymentStatusPending {
		return domain.ErrPaymentAlreadyComplete
	}
	p.Status = domain.PaymentStatusSuccess
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

func (m *mockPaymentRepo) MarkFailed(ctx context.Context, paymentID string) error {
	if m.markFailedFunc != nil {
		return m.markFailedFunc(ctx, paymentID)
	}
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if p.Status == domain.PaymentStatusSuccess || p.Status == domain.PaymentStatusFailed {
		return nil
	}
	p.Status = domain.PaymentStatusFailed
	return nil
}

func (m *mockPaymentRepo) MarkRefundRequired(ctx context.Context, paymentID, transactionID string, paidAt time.Time) error {
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusRefundRequired
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

type mockBookingRepo struct {
	bookings           map[uuid.UUID]*domain.Booking
	confirmPendingFunc func(ctx context.Context, id uuid.UUID) error
}

func (m *mockBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	if b, ok := m.bookings[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *mockBookingRepo) FindByBookingNumber(ctx context.Context, num string) (*domain.Booking, error) {
	return nil, nil
}

func (m *mockBookingRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}

func (m *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if b, ok := m.bookings[id]; ok {
		b.Status = status
		return nil
	}
	return domain.ErrBookingNotFound
}

func (m *mockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	if m.confirmPendingFunc != nil {
		return m.confirmPendingFunc(ctx, id)
	}
	b, ok := m.bookings[id]
	if !ok {
		return domain.ErrBookingNotFound
	}
	if b.Status != domain.BookingStatusPending {
		return domain.ErrBookingNotFound
	}
	b.Status = domain.BookingStatusConfirmed
	return nil
}

type mockSeatReservationRepo struct {
	reservations map[uuid.UUID]*domain.SeatReservation
	items        map[uuid.UUID][]domain.SeatReservationItem
}

func (m *mockSeatReservationRepo) Create(ctx context.Context, reservation *domain.SeatReservation) error {
	m.reservations[reservation.ID] = reservation
	return nil
}
func (m *mockSeatReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	m.items[item.ReservationID] = append(m.items[item.ReservationID], *item)
	return nil
}
func (m *mockSeatReservationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	if r, ok := m.reservations[id]; ok {
		return r, nil
	}
	return nil, domain.ErrSeatReservationNotFound
}
func (m *mockSeatReservationRepo) FindItemsByReservationID(ctx context.Context, resID uuid.UUID) ([]domain.SeatReservationItem, error) {
	return m.items[resID], nil
}
func (m *mockSeatReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if r, ok := m.reservations[id]; ok {
		r.Status = status
		return nil
	}
	return domain.ErrSeatReservationNotFound
}
func (m *mockSeatReservationRepo) CompleteIfActive(ctx context.Context, id uuid.UUID) (bool, error) {
	r, ok := m.reservations[id]
	if !ok || r.Status != domain.ReservationStatusActive {
		return false, nil
	}
	r.Status = domain.ReservationStatusCompleted
	return true, nil
}
func (m *mockSeatReservationRepo) ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	r, ok := m.reservations[id]
	if !ok || r.Status != domain.ReservationStatusActive || r.ExpiresAt.After(now) {
		return false, nil
	}
	r.Status = domain.ReservationStatusExpired
	return true, nil
}
func (m *mockSeatReservationRepo) FindExpiredActive(ctx context.Context, now time.Time, limit int) ([]domain.SeatReservation, error) {
	var list []domain.SeatReservation
	for _, r := range m.reservations {
		if r.Status == domain.ReservationStatusActive && !r.ExpiresAt.After(now) {
			list = append(list, *r)
		}
	}
	return list, nil
}
func (m *mockSeatReservationRepo) CancelIfActive(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	r, ok := m.reservations[id]
	if !ok || r.Status != domain.ReservationStatusActive {
		return false, nil
	}
	r.Status = domain.ReservationStatusCancelled
	return true, nil
}
func (m *mockSeatReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	return m.FindByID(ctx, id)
}

type mockSeatRepo struct {
	seats map[uuid.UUID]*domain.Seat
}

func (m *mockSeatRepo) Create(seat *domain.Seat) error                     { m.seats[seat.ID] = seat; return nil }
func (m *mockSeatRepo) Update(seat *domain.Seat) error                     { m.seats[seat.ID] = seat; return nil }
func (m *mockSeatRepo) Delete(id uuid.UUID) error                          { delete(m.seats, id); return nil }
func (m *mockSeatRepo) DeleteByRowID(rowID uuid.UUID) error                { return nil }
func (m *mockSeatRepo) FindByRowID(rowID uuid.UUID) ([]domain.Seat, error) { return nil, nil }
func (m *mockSeatRepo) FindByID(seatID uuid.UUID) (*domain.Seat, error) {
	if s, ok := m.seats[seatID]; ok {
		return s, nil
	}
	return nil, domain.ErrSeatNotFound
}
func (m *mockSeatRepo) CountBookedByLayoutID(layoutID uuid.UUID) (int64, error) { return 0, nil }
func (m *mockSeatRepo) ReserveIfAvailable(ctx context.Context, seatID uuid.UUID) error {
	s, ok := m.seats[seatID]
	if !ok || s.Status != domain.SeatStatusAvailable {
		return domain.ErrSeatNotAvailable
	}
	s.Status = domain.SeatStatusReserved
	return nil
}
func (m *mockSeatRepo) ReleaseReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	s, ok := m.seats[seatID]
	if !ok || s.Status != domain.SeatStatusReserved {
		return domain.ErrSeatNotAvailable
	}
	s.Status = domain.SeatStatusAvailable
	return nil
}
func (m *mockSeatRepo) BookReservedSeat(ctx context.Context, seatID uuid.UUID) error {
	s, ok := m.seats[seatID]
	if !ok || s.Status != domain.SeatStatusReserved {
		return domain.ErrSeatNotAvailable
	}
	s.Status = domain.SeatStatusBooked
	return nil
}

type mockCheckoutRepo struct {
	checkouts       map[uuid.UUID]*domain.Checkout
	updateStatusErr error
	findByResErr    error
}

func (m *mockCheckoutRepo) Create(ctx context.Context, checkout *domain.Checkout) error {
	m.checkouts[checkout.ID] = checkout
	return nil
}
func (m *mockCheckoutRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Checkout, error) {
	if c, ok := m.checkouts[id]; ok {
		return c, nil
	}
	return nil, domain.ErrCheckoutNotFound
}
func (m *mockCheckoutRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Checkout, error) {
	if m.findByResErr != nil {
		return nil, m.findByResErr
	}
	for _, c := range m.checkouts {
		if c.ReservationID == resID {
			return c, nil
		}
	}
	return nil, domain.ErrCheckoutNotFound
}
func (m *mockCheckoutRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if c, ok := m.checkouts[id]; ok {
		c.Status = status
		return nil
	}
	return domain.ErrCheckoutNotFound
}
func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, resID uuid.UUID) (bool, error) {
	for _, c := range m.checkouts {
		if c.ReservationID == resID && c.Status == domain.CheckoutStatusActive {
			c.Status = domain.CheckoutStatusExpired
			return true, nil
		}
	}
	return false, nil
}

type mockTxRepos struct {
	domain.TransactionRepositories
	paymentRepo         *mockPaymentRepo
	bookingRepo         *mockBookingRepo
	webhookRepo         *mockWebhookEventRepo
	seatReservationRepo *mockSeatReservationRepo
	seatRepo            *mockSeatRepo
	checkoutRepo        *mockCheckoutRepo
}

func (r *mockTxRepos) PaymentRepository() domain.PaymentRepository {
	return r.paymentRepo
}

func (r *mockTxRepos) BookingRepository() domain.BookingRepository {
	return r.bookingRepo
}

func (r *mockTxRepos) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository {
	return r.webhookRepo
}

func (r *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return r.seatReservationRepo
}

func (r *mockTxRepos) SeatRepository() domain.SeatRepository {
	return r.seatRepo
}

func (r *mockTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return r.checkoutRepo
}

type mockTxManager struct {
	repos *mockTxRepos
	err   error
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(m.repos)
}

func setupPaymentUsecaseTest() (*paymentuc.PaymentUsecase, *mockTxRepos, *mockPaymentProvider) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	paymentRepo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	bookingRepo := &mockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)}
	webhookRepo := &mockWebhookEventRepo{}
	seatReservationRepo := &mockSeatReservationRepo{
		reservations: make(map[uuid.UUID]*domain.SeatReservation),
		items:        make(map[uuid.UUID][]domain.SeatReservationItem),
	}
	seatRepo := &mockSeatRepo{seats: make(map[uuid.UUID]*domain.Seat)}
	checkoutRepo := &mockCheckoutRepo{checkouts: make(map[uuid.UUID]*domain.Checkout)}

	txRepos := &mockTxRepos{
		paymentRepo:         paymentRepo,
		bookingRepo:         bookingRepo,
		webhookRepo:         webhookRepo,
		seatReservationRepo: seatReservationRepo,
		seatRepo:            seatRepo,
		checkoutRepo:        checkoutRepo,
	}

	txManager := &mockTxManager{repos: txRepos}
	provider := &mockPaymentProvider{}

	uc := paymentuc.NewPaymentUsecase(txManager, nil, provider, logger)
	return uc, txRepos, provider
}

func makeWebhookPayload(event, orderID, paymentID string, amountPaise int64, currency string) []byte {
	m := map[string]interface{}{
		"entity": "event",
		"event":  event,
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       paymentID,
					"order_id": orderID,
					"amount":   amountPaise,
					"currency": currency,
					"status":   "captured",
				},
			},
		},
	}
	b, _ := json.Marshal(m)
	return b
}

func TestProcessRazorpayWebhook_InvalidSignature(t *testing.T) {
	uc, _, provider := setupPaymentUsecaseTest()
	provider.verifyWebhookSignatureFunc = func(body []byte, sig string) bool {
		return false
	}

	err := uc.ProcessRazorpayWebhook(context.Background(), []byte(`{}`), "invalid_sig", "evt_123")
	assert.ErrorIs(t, err, domain.ErrInvalidPayment)
}

func TestProcessRazorpayWebhook_InvalidJSON(t *testing.T) {
	uc, _, _ := setupPaymentUsecaseTest()
	err := uc.ProcessRazorpayWebhook(context.Background(), []byte(`{invalid-json`), "sig", "evt_123")
	assert.ErrorIs(t, err, domain.ErrInvalidPayment)
}

func TestProcessRazorpayWebhook_MissingEventID(t *testing.T) {
	uc, _, _ := setupPaymentUsecaseTest()
	body := []byte(`{"entity":"event","event":"payment.captured"}`)
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "")
	assert.ErrorIs(t, err, domain.ErrInvalidPayment)
}

func TestProcessRazorpayWebhook_FallbackToPayloadIDWhenHeaderEmpty(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_test_999"
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	body := []byte(`{
		"entity": "event",
		"id": "evt_fallback_123",
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "pay_rzp_123",
					"order_id": "order_test_999",
					"amount": 50000,
					"currency": "INR",
					"status": "captured"
				}
			}
		}
	}`)

	// Pass empty header eventID to test fallback
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "")
	require.NoError(t, err)
	assert.Contains(t, txRepos.webhookRepo.created, "evt_fallback_123")
}

func TestProcessRazorpayWebhook_DuplicateEvent_IgnoredCleanly(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	txRepos.webhookRepo.createFunc = func(ctx context.Context, eventID, eventType string) error {
		return domain.ErrDuplicateWebhookEvent
	}

	body := makeWebhookPayload("payment.captured", "order_123", "pay_rzp_123", 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_duplicate")
	require.NoError(t, err) // Duplicate must return nil / HTTP 200
}

func TestProcessRazorpayWebhook_PaymentCaptured_Success(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_captured_1"
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(350),
		Status:          domain.PaymentStatusPending,
	}

	body := makeWebhookPayload("payment.captured", orderID, "pay_rzp_captured", 35000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_captured_1")
	require.NoError(t, err)

	payment := txRepos.paymentRepo.payments["pay_local_1"]
	assert.Equal(t, domain.PaymentStatusSuccess, payment.Status)
	assert.NotNil(t, payment.TransactionID)
	assert.Equal(t, "pay_rzp_captured", *payment.TransactionID)

	booking := txRepos.bookingRepo.bookings[bookingID]
	assert.Equal(t, domain.BookingStatusConfirmed, booking.Status)
}

func TestProcessRazorpayWebhook_PaymentCaptured_IdempotentAlreadySuccess(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_already_success"
	txID := "pay_rzp_same"
	now := time.Now()

	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusConfirmed,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		TransactionID:   &txID,
		Amount:          decimal.NewFromInt(100),
		Status:          domain.PaymentStatusSuccess,
		PaidAt:          &now,
	}

	body := makeWebhookPayload("payment.captured", orderID, txID, 10000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_already_done")
	require.NoError(t, err)
}

func TestProcessRazorpayWebhook_PaymentCaptured_AlreadySuccessDifferentTxID(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_conflict"
	existingTxID := "pay_rzp_old"
	now := time.Now()

	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusConfirmed,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		TransactionID:   &existingTxID,
		Amount:          decimal.NewFromInt(100),
		Status:          domain.PaymentStatusSuccess,
		PaidAt:          &now,
	}

	body := makeWebhookPayload("payment.captured", orderID, "pay_rzp_new_different", 10000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_conflict")
	assert.ErrorIs(t, err, domain.ErrPaymentAlreadyComplete)
}

func TestProcessRazorpayWebhook_PaymentCaptured_AmountMismatch(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_mismatch"
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	// Payload sends 40000 paise (400 INR) instead of 50000 paise (500 INR)
	body := makeWebhookPayload("payment.captured", orderID, "pay_rzp_123", 40000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_mismatch")
	assert.ErrorIs(t, err, domain.ErrPaymentAmountMismatch)
}

func TestProcessRazorpayWebhook_PaymentCaptured_CurrencyMismatch(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	orderID := "order_currency_bad"
	body := makeWebhookPayload("payment.captured", orderID, "pay_rzp_123", 50000, "USD")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_currency_bad")
	assert.ErrorIs(t, err, domain.ErrInvalidPayment)
	_ = txRepos
}

func TestProcessRazorpayWebhook_PaymentCaptured_OrderNotFound(t *testing.T) {
	uc, _, _ := setupPaymentUsecaseTest()
	body := makeWebhookPayload("payment.captured", "nonexistent_order", "pay_rzp_123", 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_not_found")
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestProcessRazorpayWebhook_PaymentFailed_PendingPayment_MarkedFailed(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_failed_1"
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(250),
		Status:          domain.PaymentStatusPending,
	}

	body := makeWebhookPayload("payment.failed", orderID, "pay_rzp_failed", 25000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_failed_1")
	require.NoError(t, err)

	payment := txRepos.paymentRepo.payments["pay_local_1"]
	assert.Equal(t, domain.PaymentStatusFailed, payment.Status)

	// Decision 1: Booking remains PENDING on payment failure
	booking := txRepos.bookingRepo.bookings[bookingID]
	assert.Equal(t, domain.BookingStatusPending, booking.Status)
}

func TestProcessRazorpayWebhook_PaymentFailed_AlreadySuccess_NeverOverwritten(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_already_success"
	txID := "pay_rzp_captured"
	now := time.Now()

	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusConfirmed,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		TransactionID:   &txID,
		Amount:          decimal.NewFromInt(250),
		Status:          domain.PaymentStatusSuccess,
		PaidAt:          &now,
	}

	// Delayed payment.failed event arrives after successful capture
	body := makeWebhookPayload("payment.failed", orderID, "pay_rzp_declined_attempt", 25000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_failed_delayed")
	require.NoError(t, err)

	// Decision 6: Never allow delayed failure events to overwrite successful payments
	payment := txRepos.paymentRepo.payments["pay_local_1"]
	assert.Equal(t, domain.PaymentStatusSuccess, payment.Status)

	booking := txRepos.bookingRepo.bookings[bookingID]
	assert.Equal(t, domain.BookingStatusConfirmed, booking.Status)
}

func TestProcessRazorpayWebhook_PaymentFailed_AlreadyFailed_SafeNoOp(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	bookingID := uuid.New()
	orderID := "order_already_failed"
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(250),
		Status:          domain.PaymentStatusFailed,
	}

	body := makeWebhookPayload("payment.failed", orderID, "pay_rzp_failed_again", 25000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_failed_retry")
	require.NoError(t, err)

	payment := txRepos.paymentRepo.payments["pay_local_1"]
	assert.Equal(t, domain.PaymentStatusFailed, payment.Status)
}

func TestProcessRazorpayWebhook_UnhandledEvent_IgnoredCleanly(t *testing.T) {
	uc, _, _ := setupPaymentUsecaseTest()
	body := makeWebhookPayload("order.paid", "order_unhandled", "pay_123", 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_unhandled")
	require.NoError(t, err)
}

func TestProcessRazorpayWebhook_DatabaseError_Propagated(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	txRepos.webhookRepo.createFunc = func(ctx context.Context, eventID, eventType string) error {
		return errors.New("db connection failure")
	}

	body := makeWebhookPayload("payment.captured", "order_123", "pay_123", 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_dberr")
	assert.ErrorContains(t, err, "db connection failure")
}

func TestPaymentUsecase_VerifyPayment_SeatBooking_Success(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	reservationID := uuid.New()
	seatID := uuid.New()
	bookingID := uuid.New()
	orderID := "order_seat_success"
	paymentID := "pay_local_seat_1"
	rzpPaymentID := "pay_rzp_seat_123"

	// 1. Active reservation with reserved seat
	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	txRepos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			SeatID:        &seatID,
		},
	}
	txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusReserved,
	}

	// 2. Active checkout
	checkoutID := uuid.New()
	txRepos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		Status:        domain.CheckoutStatusActive,
	}

	// 3. Pending booking and payment
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
		Total:         decimal.NewFromInt(1000),
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(1000),
		Status:          domain.PaymentStatusPending,
	}

	// Execute verify payment
	res, err := uc.VerifyPayment(context.Background(), paymentuc.VerifyPaymentRequest{
		UserID:            userID,
		ProviderOrderID:   orderID,
		RazorpayPaymentID: rzpPaymentID,
		RazorpaySignature: "valid_signature",
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Verify all inventory lifecycle transitions
	assert.Equal(t, domain.PaymentStatusSuccess, res.Payment.Status)
	assert.Equal(t, domain.BookingStatusConfirmed, res.Booking.Status)
	assert.Equal(t, domain.ReservationStatusCompleted, txRepos.seatReservationRepo.reservations[reservationID].Status)
	assert.Equal(t, domain.SeatStatusBooked, txRepos.seatRepo.seats[seatID].Status)
	assert.Equal(t, domain.CheckoutStatusCompleted, txRepos.checkoutRepo.checkouts[checkoutID].Status)
}

func TestPaymentUsecase_VerifyPayment_GeneralAdmission_Success(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	reservationID := uuid.New()
	ticketTypeID := uuid.New()
	bookingID := uuid.New()
	orderID := "order_ga_success"
	paymentID := "pay_local_ga_1"
	rzpPaymentID := "pay_rzp_ga_123"

	// 1. Active reservation with GA ticket tier
	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	txRepos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			TicketTypeID:  &ticketTypeID,
			Quantity:      2,
		},
	}

	// 2. Pending booking and payment
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
		Total:         decimal.NewFromInt(500),
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	res, err := uc.VerifyPayment(context.Background(), paymentuc.VerifyPaymentRequest{
		UserID:            userID,
		ProviderOrderID:   orderID,
		RazorpayPaymentID: rzpPaymentID,
		RazorpaySignature: "valid_signature",
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, domain.PaymentStatusSuccess, res.Payment.Status)
	assert.Equal(t, domain.BookingStatusConfirmed, res.Booking.Status)
	assert.Equal(t, domain.ReservationStatusCompleted, txRepos.seatReservationRepo.reservations[reservationID].Status)
}

func TestPaymentUsecase_VerifyPayment_LateCapture_ExpiredReservation_StrictRejection(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	reservationID := uuid.New()
	seatID := uuid.New()
	bookingID := uuid.New()
	orderID := "order_late_verify"
	paymentID := "pay_local_late_1"
	rzpPaymentID := "pay_rzp_late_123"

	// 1. Reservation has already expired
	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		Status:    domain.ReservationStatusExpired,
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	txRepos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			SeatID:        &seatID,
		},
	}
	// The seat was released to AVAILABLE by the expiry worker
	txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusAvailable,
	}

	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
		Total:         decimal.NewFromInt(750),
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(750),
		Status:          domain.PaymentStatusPending,
	}

	// Verification attempt on expired reservation
	res, err := uc.VerifyPayment(context.Background(), paymentuc.VerifyPaymentRequest{
		UserID:            userID,
		ProviderOrderID:   orderID,
		RazorpayPaymentID: rzpPaymentID,
		RazorpaySignature: "valid_signature",
	})
	assert.ErrorIs(t, err, domain.ErrReservationExpired)
	assert.Nil(t, res)

	// Strict rejection verification:
	payment := txRepos.paymentRepo.payments[paymentID]
	booking := txRepos.bookingRepo.bookings[bookingID]
	seat := txRepos.seatRepo.seats[seatID]

	// Payment must be REFUND_REQUIRED, with transaction ID recorded
	assert.Equal(t, domain.PaymentStatusRefundRequired, payment.Status)
	assert.NotNil(t, payment.TransactionID)
	assert.Equal(t, rzpPaymentID, *payment.TransactionID)

	// Booking must be CANCELLED (never confirmed!)
	assert.Equal(t, domain.BookingStatusCancelled, booking.Status)

	// Seat must NOT be booked
	assert.Equal(t, domain.SeatStatusAvailable, seat.Status)
}

func TestPaymentUsecase_Webhook_Captured_LatePayment_ExpiredReservation(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	reservationID := uuid.New()
	bookingID := uuid.New()
	orderID := "order_wh_late"
	paymentID := "pay_local_wh_late"
	rzpPaymentID := "pay_rzp_wh_late_456"

	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		Status:    domain.ReservationStatusExpired,
		ExpiresAt: time.Now().Add(-2 * time.Minute),
	}
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	body := makeWebhookPayload("payment.captured", orderID, rzpPaymentID, 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_wh_late_1")
	// Webhook must acknowledge with 200 OK (nil error) so Razorpay doesn't retry
	require.NoError(t, err)

	payment := txRepos.paymentRepo.payments[paymentID]
	booking := txRepos.bookingRepo.bookings[bookingID]

	assert.Equal(t, domain.PaymentStatusRefundRequired, payment.Status)
	assert.Equal(t, domain.BookingStatusCancelled, booking.Status)
}

func TestPaymentUsecase_Webhook_DuplicateCaptured_OnRefundRequiredPayment(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	reservationID := uuid.New()
	bookingID := uuid.New()
	orderID := "order_dup_refund"
	paymentID := "pay_local_dup_refund"
	rzpPaymentID := "pay_rzp_dup_789"

	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:     reservationID,
		Status: domain.ReservationStatusExpired,
	}
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusCancelled,
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusRefundRequired,
		TransactionID:   &rzpPaymentID,
	}

	body := makeWebhookPayload("payment.captured", orderID, rzpPaymentID, 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_dup_refund_1")
	require.NoError(t, err)

	// Statuses remain intact
	assert.Equal(t, domain.PaymentStatusRefundRequired, txRepos.paymentRepo.payments[paymentID].Status)
	assert.Equal(t, domain.BookingStatusCancelled, txRepos.bookingRepo.bookings[bookingID].Status)
}

func TestPaymentUsecase_VerifyPayment_ConcurrentOnRefundRequired(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	orderID := "order_verify_concurrent_refund"
	paymentID := "pay_local_conc_refund"
	rzpPaymentID := "pay_rzp_conc_111"
	bookingID := uuid.New()

	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		Status: domain.BookingStatusCancelled,
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(300),
		Status:          domain.PaymentStatusRefundRequired,
		TransactionID:   &rzpPaymentID,
	}

	res, err := uc.VerifyPayment(context.Background(), paymentuc.VerifyPaymentRequest{
		UserID:            userID,
		ProviderOrderID:   orderID,
		RazorpayPaymentID: rzpPaymentID,
		RazorpaySignature: "sig",
	})
	assert.ErrorIs(t, err, domain.ErrReservationExpired)
	assert.Nil(t, res)
}

func TestPaymentUsecase_ExpiryWorker_CannotReleaseCompletedReservation(t *testing.T) {
	_, txRepos, _ := setupPaymentUsecaseTest()

	reservationID := uuid.New()
	seatID := uuid.New()

	// Completed reservation with BOOKED seat
	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		Status:    domain.ReservationStatusCompleted,
		ExpiresAt: time.Now().Add(-10 * time.Minute), // Past expiry time!
	}
	txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusBooked,
	}

	// ExpireIfActive called by worker
	expired, err := txRepos.seatReservationRepo.ExpireIfActive(context.Background(), reservationID, time.Now())
	require.NoError(t, err)
	assert.False(t, expired, "worker must not expire a COMPLETED reservation")

	// Seat remains BOOKED
	assert.Equal(t, domain.SeatStatusBooked, txRepos.seatRepo.seats[seatID].Status)
	assert.Equal(t, domain.ReservationStatusCompleted, txRepos.seatReservationRepo.reservations[reservationID].Status)
}

func TestPaymentUsecase_VerifyPayment_CheckoutUpdateFailure_RollsBack(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	reservationID := uuid.New()
	seatID := uuid.New()
	bookingID := uuid.New()
	checkoutID := uuid.New()
	orderID := "order_checkout_fail"
	paymentID := "pay_checkout_fail_1"
	rzpPaymentID := "pay_rzp_chk_123"

	// 1. Active reservation with reserved seat
	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	txRepos.seatReservationRepo.items[reservationID] = []domain.SeatReservationItem{
		{
			ID:            uuid.New(),
			ReservationID: reservationID,
			SeatID:        &seatID,
		},
	}
	txRepos.seatRepo.seats[seatID] = &domain.Seat{
		ID:     seatID,
		Status: domain.SeatStatusReserved,
	}

	// 2. Active checkout
	txRepos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		Status:        domain.CheckoutStatusActive,
	}

	// 3. Pending booking and payment
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
		Total:         decimal.NewFromInt(1000),
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(1000),
		Status:          domain.PaymentStatusPending,
	}

	// Simulate checkout update failure
	checkoutErr := errors.New("database failure updating checkout status")
	txRepos.checkoutRepo.updateStatusErr = checkoutErr

	res, err := uc.VerifyPayment(context.Background(), paymentuc.VerifyPaymentRequest{
		UserID:            userID,
		ProviderOrderID:   orderID,
		RazorpayPaymentID: rzpPaymentID,
		RazorpaySignature: "valid_signature",
	})
	assert.ErrorIs(t, err, checkoutErr)
	assert.Nil(t, res)

	// Verify that payment and booking were NOT confirmed
	assert.Equal(t, domain.PaymentStatusPending, txRepos.paymentRepo.payments[paymentID].Status, "payment must not be marked SUCCESS")
	assert.Equal(t, domain.BookingStatusPending, txRepos.bookingRepo.bookings[bookingID].Status, "booking must not be marked CONFIRMED")
	assert.Equal(t, domain.CheckoutStatusActive, txRepos.checkoutRepo.checkouts[checkoutID].Status, "checkout must not be marked COMPLETED")
}

func TestPaymentUsecase_Webhook_Captured_CheckoutUpdateFailure_RollsBack(t *testing.T) {
	uc, txRepos, _ := setupPaymentUsecaseTest()

	userID := uuid.New()
	reservationID := uuid.New()
	bookingID := uuid.New()
	checkoutID := uuid.New()
	orderID := "order_wh_chk_fail"
	paymentID := "pay_wh_chk_fail_1"
	rzpPaymentID := "pay_rzp_wh_chk_123"

	txRepos.seatReservationRepo.reservations[reservationID] = &domain.SeatReservation{
		ID:        reservationID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	txRepos.checkoutRepo.checkouts[checkoutID] = &domain.Checkout{
		ID:            checkoutID,
		ReservationID: reservationID,
		Status:        domain.CheckoutStatusActive,
	}
	txRepos.bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		ReservationID: &reservationID,
		Status:        domain.BookingStatusPending,
		Total:         decimal.NewFromInt(500),
	}
	txRepos.paymentRepo.payments[paymentID] = &domain.Payment{
		ID:              paymentID,
		BookingID:       bookingID.String(),
		Provider:        domain.PaymentProviderRazorpay,
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	checkoutErr := errors.New("checkout db update error")
	txRepos.checkoutRepo.updateStatusErr = checkoutErr

	body := makeWebhookPayload("payment.captured", orderID, rzpPaymentID, 50000, "INR")
	err := uc.ProcessRazorpayWebhook(context.Background(), body, "sig", "evt_chk_fail_1")
	assert.ErrorIs(t, err, checkoutErr)

	// Verify that payment and booking were NOT confirmed
	assert.Equal(t, domain.PaymentStatusPending, txRepos.paymentRepo.payments[paymentID].Status, "payment must not be marked SUCCESS")
	assert.Equal(t, domain.BookingStatusPending, txRepos.bookingRepo.bookings[bookingID].Status, "booking must not be marked CONFIRMED")
}

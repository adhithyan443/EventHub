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

type mockTxRepos struct {
	domain.TransactionRepositories
	paymentRepo *mockPaymentRepo
	bookingRepo *mockBookingRepo
	webhookRepo *mockWebhookEventRepo
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

	txRepos := &mockTxRepos{
		paymentRepo: paymentRepo,
		bookingRepo: bookingRepo,
		webhookRepo: webhookRepo,
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

package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	paymentuc "github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handlerMockPaymentProvider struct {
	webhookSecret string
}

func (m *handlerMockPaymentProvider) Initiate(ctx context.Context, amount decimal.Decimal, currency string, idempotencyKey string) (string, error) {
	return "order_123", nil
}

func (m *handlerMockPaymentProvider) Verify(ctx context.Context, providerOrderID, paymentID, signature string, expectedAmount decimal.Decimal, currency string) (bool, error) {
	return true, nil
}

func (m *handlerMockPaymentProvider) VerifyWebhookSignature(body []byte, signature string) bool {
	if len(body) == 0 || strings.TrimSpace(signature) == "" {
		return false
	}
	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(m.webhookSecret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), providedSignature)
}

type handlerMockWebhookRepo struct {
	createFunc func(ctx context.Context, eventID, eventType string) error
}

func (m *handlerMockWebhookRepo) Create(ctx context.Context, eventID, eventType string) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, eventID, eventType)
	}
	return nil
}

type handlerMockPaymentRepo struct {
	payments           map[string]*domain.Payment
	markSuccessfulFunc func(ctx context.Context, paymentID, transactionID string, paidAt time.Time) error
	markFailedFunc     func(ctx context.Context, paymentID string) error
}

func (m *handlerMockPaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *handlerMockPaymentRepo) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *handlerMockPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	return nil, nil
}

func (m *handlerMockPaymentRepo) Update(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *handlerMockPaymentRepo) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Payment, error) {
	for _, p := range m.payments {
		if p.ProviderOrderID != nil && *p.ProviderOrderID == providerOrderID {
			return p, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *handlerMockPaymentRepo) MarkSuccessful(ctx context.Context, paymentID, transactionID string, paidAt time.Time) error {
	if m.markSuccessfulFunc != nil {
		return m.markSuccessfulFunc(ctx, paymentID, transactionID, paidAt)
	}
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusSuccess
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

func (m *handlerMockPaymentRepo) MarkFailed(ctx context.Context, paymentID string) error {
	if m.markFailedFunc != nil {
		return m.markFailedFunc(ctx, paymentID)
	}
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusFailed
	return nil
}

func (m *handlerMockPaymentRepo) MarkRefundRequired(ctx context.Context, paymentID, transactionID string, paidAt time.Time) error {
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusRefundRequired
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

type handlerMockBookingRepo struct {
	bookings map[uuid.UUID]*domain.Booking
}

func (m *handlerMockBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	return nil
}

func (m *handlerMockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	if b, ok := m.bookings[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *handlerMockBookingRepo) FindByBookingNumber(ctx context.Context, num string) (*domain.Booking, error) {
	return nil, nil
}

func (m *handlerMockBookingRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}

func (m *handlerMockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

func (m *handlerMockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	b, ok := m.bookings[id]
	if !ok {
		return domain.ErrBookingNotFound
	}
	b.Status = domain.BookingStatusConfirmed
	return nil
}

type handlerMockTxRepos struct {
	domain.TransactionRepositories
	paymentRepo *handlerMockPaymentRepo
	bookingRepo *handlerMockBookingRepo
	webhookRepo *handlerMockWebhookRepo
}

func (r *handlerMockTxRepos) PaymentRepository() domain.PaymentRepository {
	return r.paymentRepo
}

func (r *handlerMockTxRepos) BookingRepository() domain.BookingRepository {
	return r.bookingRepo
}

func (r *handlerMockTxRepos) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository {
	return r.webhookRepo
}

type handlerMockTxManager struct {
	repos *handlerMockTxRepos
	err   error
}

func (m *handlerMockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(m.repos)
}

func setupWebhookTestRouter(
	secret string,
	txRepos *handlerMockTxRepos,
	txManager *handlerMockTxManager,
) (*gin.Engine, *handler.PaymentHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	provider := &handlerMockPaymentProvider{webhookSecret: secret}
	usecase := paymentuc.NewPaymentUsecase(txManager, nil, provider, logger)
	paymentHandler := handler.NewPaymentHandler(usecase, "rzp_test_key", logger)

	api := router.Group("/api/v1")
	api.POST("/payments/webhook", paymentHandler.RazorpayWebhook)

	return router, paymentHandler
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPaymentHandler_RazorpayWebhook_Success(t *testing.T) {
	secret := "whsec_test_secret"
	bookingID := uuid.New()
	orderID := "order_wh_1"

	paymentRepo := &handlerMockPaymentRepo{
		payments: map[string]*domain.Payment{
			"pay_1": {
				ID:              "pay_1",
				BookingID:       bookingID.String(),
				ProviderOrderID: &orderID,
				Amount:          decimal.NewFromInt(500),
				Status:          domain.PaymentStatusPending,
			},
		},
	}
	bookingRepo := &handlerMockBookingRepo{
		bookings: map[uuid.UUID]*domain.Booking{
			bookingID: {
				ID:     bookingID,
				Status: domain.BookingStatusPending,
			},
		},
	}
	webhookRepo := &handlerMockWebhookRepo{}

	txRepos := &handlerMockTxRepos{
		paymentRepo: paymentRepo,
		bookingRepo: bookingRepo,
		webhookRepo: webhookRepo,
	}
	txManager := &handlerMockTxManager{repos: txRepos}

	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	payload := map[string]interface{}{
		"entity": "event",
		"event":  "payment.captured",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       "pay_rzp_wh_1",
					"order_id": orderID,
					"amount":   50000,
					"currency": "INR",
					"status":   "captured",
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	signature := signBody(bodyBytes, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_wh_123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "webhook processed")
}

func TestPaymentHandler_RazorpayWebhook_MissingSignature(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader([]byte(`{"test":1}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid webhook signature")
}

func TestPaymentHandler_RazorpayWebhook_InvalidSignature(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader([]byte(`{"test":1}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", "wrong_signature_hex")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPaymentHandler_RazorpayWebhook_DuplicateEvent_Returns200(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{
			createFunc: func(ctx context.Context, eventID, eventType string) error {
				return domain.ErrDuplicateWebhookEvent
			},
		},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	bodyBytes := []byte(`{"entity":"event","event":"payment.captured"}`)
	signature := signBody(bodyBytes, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_duplicate_header")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaymentHandler_RazorpayWebhook_OrderNotFound_Returns404(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	payload := map[string]interface{}{
		"entity": "event",
		"event":  "payment.captured",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       "pay_unknown",
					"order_id": "order_unknown",
					"amount":   50000,
					"currency": "INR",
					"status":   "captured",
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	signature := signBody(bodyBytes, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_unknown")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentHandler_RazorpayWebhook_AmountMismatch_Returns400(t *testing.T) {
	secret := "whsec_test_secret"
	bookingID := uuid.New()
	orderID := "order_wh_mismatch"

	paymentRepo := &handlerMockPaymentRepo{
		payments: map[string]*domain.Payment{
			"pay_1": {
				ID:              "pay_1",
				BookingID:       bookingID.String(),
				ProviderOrderID: &orderID,
				Amount:          decimal.NewFromInt(500),
				Status:          domain.PaymentStatusPending,
			},
		},
	}
	txRepos := &handlerMockTxRepos{
		paymentRepo: paymentRepo,
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	payload := map[string]interface{}{
		"entity": "event",
		"event":  "payment.captured",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       "pay_rzp_1",
					"order_id": orderID,
					"amount":   99999, // Mismatched amount
					"currency": "INR",
					"status":   "captured",
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	signature := signBody(bodyBytes, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_mismatch")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "payment amount mismatch")
}

func TestPaymentHandler_RazorpayWebhook_DatabaseError_Returns500(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{
			createFunc: func(ctx context.Context, eventID, eventType string) error {
				return errors.New("db error")
			},
		},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	bodyBytes := []byte(`{"entity":"event","event":"payment.captured"}`)
	signature := signBody(bodyBytes, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_dberr")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPaymentHandler_RazorpayWebhook_BodyLimit(t *testing.T) {
	secret := "whsec_test_secret"
	txRepos := &handlerMockTxRepos{
		paymentRepo: &handlerMockPaymentRepo{payments: make(map[string]*domain.Payment)},
		bookingRepo: &handlerMockBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)},
		webhookRepo: &handlerMockWebhookRepo{},
	}
	txManager := &handlerMockTxManager{repos: txRepos}
	router, _ := setupWebhookTestRouter(secret, txRepos, txManager)

	// Body larger than 1 MB
	hugeBody := bytes.Repeat([]byte("a"), (1<<20)+100)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(hugeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", "any_sig")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid webhook body")
}

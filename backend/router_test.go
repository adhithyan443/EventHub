package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	paymentuc "github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routerTestPaymentProvider struct {
	secret string
}

func (m *routerTestPaymentProvider) Initiate(ctx context.Context, amount decimal.Decimal, currency string, idempotencyKey string) (string, error) {
	return "order_router_test", nil
}

func (m *routerTestPaymentProvider) Verify(ctx context.Context, providerOrderID, paymentID, signature string, expectedAmount decimal.Decimal, currency string) (bool, error) {
	return true, nil
}

func (m *routerTestPaymentProvider) VerifyWebhookSignature(body []byte, signature string) bool {
	if len(body) == 0 || strings.TrimSpace(signature) == "" {
		return false
	}
	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(m.secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), providedSignature)
}

type routerTestWebhookRepo struct{}

func (m *routerTestWebhookRepo) Create(ctx context.Context, eventID, eventType string) error {
	return nil
}

type routerTestPaymentRepo struct {
	payments map[string]*domain.Payment
}

func (m *routerTestPaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *routerTestPaymentRepo) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *routerTestPaymentRepo) FindByBookingID(ctx context.Context, bookingID string) ([]domain.Payment, error) {
	return nil, nil
}

func (m *routerTestPaymentRepo) Update(ctx context.Context, payment *domain.Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *routerTestPaymentRepo) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Payment, error) {
	for _, p := range m.payments {
		if p.ProviderOrderID != nil && *p.ProviderOrderID == providerOrderID {
			return p, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *routerTestPaymentRepo) MarkSuccessful(ctx context.Context, paymentID, transactionID string, paidAt time.Time) error {
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusSuccess
	p.TransactionID = &transactionID
	p.PaidAt = &paidAt
	return nil
}

func (m *routerTestPaymentRepo) MarkFailed(ctx context.Context, paymentID string) error {
	p, ok := m.payments[paymentID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	p.Status = domain.PaymentStatusFailed
	return nil
}

type routerTestBookingRepo struct {
	bookings map[uuid.UUID]*domain.Booking
}

func (m *routerTestBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	return nil
}

func (m *routerTestBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	if b, ok := m.bookings[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBookingNotFound
}

func (m *routerTestBookingRepo) FindByBookingNumber(ctx context.Context, num string) (*domain.Booking, error) {
	return nil, nil
}

func (m *routerTestBookingRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}

func (m *routerTestBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

func (m *routerTestBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error {
	b, ok := m.bookings[id]
	if !ok {
		return domain.ErrBookingNotFound
	}
	b.Status = domain.BookingStatusConfirmed
	return nil
}

type routerTestTxRepos struct {
	domain.TransactionRepositories
	paymentRepo *routerTestPaymentRepo
	bookingRepo *routerTestBookingRepo
	webhookRepo *routerTestWebhookRepo
}

func (r *routerTestTxRepos) PaymentRepository() domain.PaymentRepository {
	return r.paymentRepo
}

func (r *routerTestTxRepos) BookingRepository() domain.BookingRepository {
	return r.bookingRepo
}

func (r *routerTestTxRepos) RazorpayWebhookEventRepository() domain.RazorpayWebhookEventRepository {
	return r.webhookRepo
}

type routerTestTxManager struct {
	repos *routerTestTxRepos
}

func (m *routerTestTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func setupTestRouterInstance(secret string) (*gin.Engine, *token.JWTService, *routerTestPaymentRepo, *routerTestBookingRepo) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtSecret := "test_jwt_secret_32_characters_12345"
	jwtService := token.NewJWTService(jwtSecret)

	paymentRepo := &routerTestPaymentRepo{payments: make(map[string]*domain.Payment)}
	bookingRepo := &routerTestBookingRepo{bookings: make(map[uuid.UUID]*domain.Booking)}
	webhookRepo := &routerTestWebhookRepo{}

	txRepos := &routerTestTxRepos{
		paymentRepo: paymentRepo,
		bookingRepo: bookingRepo,
		webhookRepo: webhookRepo,
	}
	txManager := &routerTestTxManager{repos: txRepos}
	provider := &routerTestPaymentProvider{secret: secret}

	paymentUsecase := paymentuc.NewPaymentUsecase(txManager, nil, provider, logger)
	paymentHandler := handler.NewPaymentHandler(paymentUsecase, logger)

	router := setupRouter(
		logger,
		&handler.AuthHandler{},
		&handler.OrganizerApplicationHandler{},
		&handler.OrganizerProfileHandler{},
		&handler.CategoryHandler{},
		&handler.EventHandler{},
		&handler.SeatLayoutHandler{},
		&handler.ReservationHandler{},
		&handler.CheckoutHandler{},
		paymentHandler,
		jwtService,
	)

	return router, jwtService, paymentRepo, bookingRepo
}

func generateTestSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestRouter_PaymentWebhook_ReachableWithoutJWT(t *testing.T) {
	secret := "whsec_router_integration_test"
	router, _, paymentRepo, bookingRepo := setupTestRouterInstance(secret)

	bookingID := uuid.New()
	orderID := "order_wh_router"
	bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusPending,
	}
	paymentRepo.payments["pay_local_1"] = &domain.Payment{
		ID:              "pay_local_1",
		BookingID:       bookingID.String(),
		ProviderOrderID: &orderID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.PaymentStatusPending,
	}

	payload := map[string]interface{}{
		"entity": "event",
		"event":  "payment.captured",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       "pay_rzp_router_1",
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

	signature := generateTestSignature(bodyBytes, secret)

	// 1. Valid webhook without any JWT Authorization header must return 200 OK
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", signature)
	req.Header.Set("X-Razorpay-Event-Id", "evt_router_1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "webhook endpoint must be reachable without JWT authorization")
	assert.Contains(t, w.Body.String(), "webhook processed")

	// 2. Webhook with invalid signature and no JWT header must return 401 with webhook signature error,
	// proving it reached the webhook handler rather than JWT middleware.
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(bodyBytes))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("X-Razorpay-Signature", "invalid_signature")
	reqInvalid.Header.Set("X-Razorpay-Event-Id", "evt_router_2")

	wInvalid := httptest.NewRecorder()
	router.ServeHTTP(wInvalid, reqInvalid)

	assert.Equal(t, http.StatusUnauthorized, wInvalid.Code)
	assert.Contains(t, wInvalid.Body.String(), "invalid webhook signature",
		"must be rejected by webhook signature validator, not JWT auth middleware")
}

func TestRouter_PaymentRoutes_ProtectedByJWT(t *testing.T) {
	secret := "whsec_router_integration_test"
	router, jwtService, _, _ := setupTestRouterInstance(secret)

	// 1. POST /api/v1/payments without Authorization header must return 401
	reqCreateNoAuth := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader([]byte(`{}`)))
	reqCreateNoAuth.Header.Set("Content-Type", "application/json")

	wCreateNoAuth := httptest.NewRecorder()
	router.ServeHTTP(wCreateNoAuth, reqCreateNoAuth)

	assert.Equal(t, http.StatusUnauthorized, wCreateNoAuth.Code)
	assert.Contains(t, wCreateNoAuth.Body.String(), "authorization header is required")

	// 2. POST /api/v1/payments/verify without Authorization header must return 401
	reqVerifyNoAuth := httptest.NewRequest(http.MethodPost, "/api/v1/payments/verify", bytes.NewReader([]byte(`{}`)))
	reqVerifyNoAuth.Header.Set("Content-Type", "application/json")

	wVerifyNoAuth := httptest.NewRecorder()
	router.ServeHTTP(wVerifyNoAuth, reqVerifyNoAuth)

	assert.Equal(t, http.StatusUnauthorized, wVerifyNoAuth.Code)
	assert.Contains(t, wVerifyNoAuth.Body.String(), "authorization header is required")

	// 3. POST /api/v1/payments with invalid Bearer token must return 401
	reqInvalidToken := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader([]byte(`{}`)))
	reqInvalidToken.Header.Set("Content-Type", "application/json")
	reqInvalidToken.Header.Set("Authorization", "Bearer invalid-jwt-token")

	wInvalidToken := httptest.NewRecorder()
	router.ServeHTTP(wInvalidToken, reqInvalidToken)

	assert.Equal(t, http.StatusUnauthorized, wInvalidToken.Code)
	assert.Contains(t, wInvalidToken.Body.String(), "invalid or expired token")

	// 4. POST /api/v1/payments with valid Bearer token passes JWT auth (reaches CreatePayment handler)
	validToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	reqValidToken := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader([]byte(`{}`)))
	reqValidToken.Header.Set("Content-Type", "application/json")
	reqValidToken.Header.Set("Authorization", "Bearer "+validToken)

	wValidToken := httptest.NewRecorder()
	router.ServeHTTP(wValidToken, reqValidToken)

	// Since body is empty {}, CreatePayment returns 400 Bad Request (validation error),
	// proving that JWT middleware succeeded and passed control to the handler.
	assert.NotEqual(t, http.StatusUnauthorized, wValidToken.Code,
		"valid JWT token must bypass authentication middleware")
	assert.Equal(t, http.StatusBadRequest, wValidToken.Code)
}

func TestRouter_HealthRoute_Public(t *testing.T) {
	router, _, _, _ := setupTestRouterInstance("any_secret")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

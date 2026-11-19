package refund_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	domainfinance "github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/refund"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockTxManager struct {
	repos *mockTxRepos
	mu    sync.Mutex
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var snapshotByID map[uuid.UUID]*domain.Refund
	var snapshotByPay map[uuid.UUID][]domain.Refund
	var snapshotByBook map[uuid.UUID][]domain.Refund

	if m.repos != nil && m.repos.refundRepo != nil {
		m.repos.refundRepo.mu.Lock()
		snapshotByID = make(map[uuid.UUID]*domain.Refund, len(m.repos.refundRepo.refundsByID))
		for k, v := range m.repos.refundRepo.refundsByID {
			cp := *v
			snapshotByID[k] = &cp
		}
		snapshotByPay = make(map[uuid.UUID][]domain.Refund, len(m.repos.refundRepo.refundsByPayment))
		for k, v := range m.repos.refundRepo.refundsByPayment {
			snapshotByPay[k] = append([]domain.Refund(nil), v...)
		}
		snapshotByBook = make(map[uuid.UUID][]domain.Refund, len(m.repos.refundRepo.refundsByBooking))
		for k, v := range m.repos.refundRepo.refundsByBooking {
			snapshotByBook[k] = append([]domain.Refund(nil), v...)
		}
		m.repos.refundRepo.mu.Unlock()
	}

	err := fn(m.repos)
	if err != nil && snapshotByID != nil {
		m.repos.refundRepo.mu.Lock()
		m.repos.refundRepo.refundsByID = snapshotByID
		m.repos.refundRepo.refundsByPayment = snapshotByPay
		m.repos.refundRepo.refundsByBooking = snapshotByBook
		m.repos.refundRepo.mu.Unlock()
	}
	return err
}

type mockTxRepos struct {
	domain.TransactionRepositories
	refundRepo  *mockRefundRepo
	paymentRepo *mockPaymentRepo
	bookingRepo *mockBookingRepo
	eventRepo   *mockEventRepo
}

func (m *mockTxRepos) RefundRepository() domain.RefundRepository {
	return m.refundRepo
}

func (m *mockTxRepos) PaymentRepository() domain.PaymentRepository {
	return m.paymentRepo
}

func (m *mockTxRepos) BookingRepository() domain.BookingRepository {
	if m.bookingRepo != nil {
		return m.bookingRepo
	}
	return nil
}

func (m *mockTxRepos) EventRepository() domain.EventRepository {
	if m.eventRepo != nil {
		return m.eventRepo
	}
	return nil
}

type mockBookingRepo struct {
	domain.BookingRepository
	mu           sync.Mutex
	bookingsByID map[uuid.UUID]*domain.Booking
}

func newMockBookingRepo() *mockBookingRepo {
	return &mockBookingRepo{
		bookingsByID: make(map[uuid.UUID]*domain.Booking),
	}
}

func (m *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.bookingsByID[id]; ok {
		copied := *b
		return &copied, nil
	}
	return nil, domain.ErrBookingNotFound
}

type mockEventRepo struct {
	domain.EventRepository
	mu         sync.Mutex
	eventsByID map[uuid.UUID]*domain.Event
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		eventsByID: make(map[uuid.UUID]*domain.Event),
	}
}

func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.eventsByID[id]; ok {
		copied := *e
		return &copied, nil
	}
	return nil, errors.New("event not found")
}

type mockFinanceUsecase struct {
	mu                     sync.Mutex
	recordRefundWithinTxFn func(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error)
	refundInputs           []domain.RecordRefundInput
}

func (m *mockFinanceUsecase) RecordPaymentCaptureWithinTx(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordPaymentCaptureInput) (*domainfinance.JournalTransaction, error) {
	return nil, nil
}
func (m *mockFinanceUsecase) RecordPaymentCapture(ctx context.Context, in domain.RecordPaymentCaptureInput) (*domainfinance.JournalTransaction, error) {
	return nil, nil
}
func (m *mockFinanceUsecase) RecordRefundWithinTx(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refundInputs = append(m.refundInputs, in)
	if m.recordRefundWithinTxFn != nil {
		return m.recordRefundWithinTxFn(ctx, txRepos, in)
	}
	return nil, nil
}
func (m *mockFinanceUsecase) RecordRefund(ctx context.Context, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error) {
	return nil, nil
}
func (m *mockFinanceUsecase) RecordEscrowCaptureWithinTx(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordEscrowCaptureInput) (*domainfinance.JournalTransaction, error) {
	return nil, nil
}
func (m *mockFinanceUsecase) RecordEscrowCapture(ctx context.Context, in domain.RecordEscrowCaptureInput) (*domainfinance.JournalTransaction, error) {
	return nil, nil
}

type mockRefundRepo struct {
	domain.RefundRepository
	mu               sync.Mutex
	refundsByID      map[uuid.UUID]*domain.Refund
	refundsByPayment map[uuid.UUID][]domain.Refund
	refundsByBooking map[uuid.UUID][]domain.Refund
	updateCalls      int
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{
		refundsByID:      make(map[uuid.UUID]*domain.Refund),
		refundsByPayment: make(map[uuid.UUID][]domain.Refund),
		refundsByBooking: make(map[uuid.UUID][]domain.Refund),
	}
}

func (m *mockRefundRepo) Create(ctx context.Context, r *domain.Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *r
	m.refundsByID[r.ID] = &copied
	m.refundsByPayment[r.PaymentID] = append(m.refundsByPayment[r.PaymentID], copied)
	m.refundsByBooking[r.BookingID] = append(m.refundsByBooking[r.BookingID], copied)
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

func (m *mockRefundRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Refund(nil), m.refundsByBooking[bookingID]...), nil
}

func (m *mockRefundRepo) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Refund(nil), m.refundsByPayment[paymentID]...), nil
}

func (m *mockRefundRepo) FindByGatewayRefundID(ctx context.Context, gatewayRefundID string) (*domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.refundsByID {
		if r.GatewayRefundID != nil && *r.GatewayRefundID == gatewayRefundID {
			copied := *r
			return &copied, nil
		}
	}
	return nil, domain.ErrRefundNotFound
}

func (m *mockRefundRepo) FindByGatewayRefundIDForUpdate(ctx context.Context, gatewayRefundID string) (*domain.Refund, error) {
	return m.FindByGatewayRefundID(ctx, gatewayRefundID)
}

func (m *mockRefundRepo) FindStuckRefunds(ctx context.Context, olderThan time.Time, limit int) ([]domain.Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Refund
	for _, r := range m.refundsByID {
		if (r.Status == domain.RefundStatusPending || r.Status == domain.RefundStatusProcessing) && !r.UpdatedAt.After(olderThan) {
			res = append(res, *r)
			if limit > 0 && len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (m *mockRefundRepo) Update(ctx context.Context, r *domain.Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	copied := *r
	m.refundsByID[r.ID] = &copied

	byPay := m.refundsByPayment[r.PaymentID]
	for i := range byPay {
		if byPay[i].ID == r.ID {
			byPay[i] = copied
			break
		}
	}
	m.refundsByPayment[r.PaymentID] = byPay

	byBook := m.refundsByBooking[r.BookingID]
	for i := range byBook {
		if byBook[i].ID == r.ID {
			byBook[i] = copied
			break
		}
	}
	m.refundsByBooking[r.BookingID] = byBook

	return nil
}

type mockPaymentRepo struct {
	domain.PaymentRepository
	mu           sync.Mutex
	paymentsByID map[string]*domain.Payment
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		paymentsByID: make(map[string]*domain.Payment),
	}
}

func (m *mockPaymentRepo) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.paymentsByID[id]; ok {
		copied := *p
		return &copied, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByIDForUpdate(ctx context.Context, id string) (*domain.Payment, error) {
	return m.FindByID(ctx, id)
}

type mockRefundProvider struct {
	mu             sync.Mutex
	refundResult   *domain.RefundResult
	refundErr      error
	fetchRefunds   []domain.RefundResult
	fetchErr       error
	refundCalls    int
	fetchCalls     int
	lastReceipt    string
	lastCurrency   string
	lastGatewayPay string
}

func (m *mockRefundProvider) Refund(
	ctx context.Context,
	gatewayPaymentID string,
	amount decimal.Decimal,
	currency string,
	idempotencyKey string,
) (*domain.RefundResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refundCalls++
	m.lastGatewayPay = gatewayPaymentID
	m.lastCurrency = currency
	m.lastReceipt = idempotencyKey
	if m.refundErr != nil {
		return nil, m.refundErr
	}
	return m.refundResult, nil
}

func (m *mockRefundProvider) FetchRefunds(
	ctx context.Context,
	gatewayPaymentID string,
) ([]domain.RefundResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchCalls++
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.fetchRefunds, nil
}

func setupRefundUsecase(t *testing.T) (*refund.RefundUsecase, *mockTxRepos, *mockRefundProvider) {
	refundRepo := newMockRefundRepo()
	paymentRepo := newMockPaymentRepo()
	bookingRepo := newMockBookingRepo()
	eventRepo := newMockEventRepo()
	txRepos := &mockTxRepos{
		refundRepo:  refundRepo,
		paymentRepo: paymentRepo,
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
	}
	txMgr := &mockTxManager{repos: txRepos}
	provider := &mockRefundProvider{}
	uc := refund.NewRefundUsecase(txMgr, provider, slog.Default())
	return uc, txRepos, provider
}

// --- Tests ---

func TestRefundUsecase_ProcessPendingRefund_Success(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	bookingID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repos.refundRepo.Create(ctx, ref)
	require.NoError(t, err)

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_abc123",
		Amount:   decimal.NewFromInt(500),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.NotNil(t, result.GatewayRefundID)
	assert.Equal(t, "rfnd_abc123", *result.GatewayRefundID)
	assert.NotNil(t, result.ProcessedAt)
	assert.Equal(t, 1, provider.refundCalls)
}

func TestRefundUsecase_ProcessPendingRefund_IdempotentAlreadySuccess(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	bookingID := uuid.New()
	rfndID := "rfnd_existing"
	now := time.Now()
	ref := &domain.Refund{
		ID:              refundID,
		BookingID:       bookingID,
		PaymentID:       paymentID,
		GatewayRefundID: &rfndID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.RefundStatusSuccess,
		ProcessedAt:     &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	err := repos.refundRepo.Create(ctx, ref)
	require.NoError(t, err)

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.Equal(t, 0, provider.refundCalls)
}

func TestRefundUsecase_ProcessPendingRefund_TotalRefundCapExceeded(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	// Existing completed refund of 800
	existingRef := &domain.Refund{
		ID:        uuid.New(),
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(800),
		Status:    domain.RefundStatusSuccess,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, existingRef)

	// New refund requests 300 (800 + 300 = 1100 > 1000 payment)
	newRefundID := uuid.New()
	newRef := &domain.Refund{
		ID:        newRefundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(300),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, newRef)

	result, err := uc.ProcessPendingRefund(ctx, newRefundID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrRefundExceedsPayment))
	assert.Nil(t, result)
	assert.Equal(t, 0, provider.refundCalls)
}

func TestRefundUsecase_ProcessPendingRefund_TwoConcurrentRefundsExceedPayment(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_concurrent_cap"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refund1ID := uuid.New()
	ref1 := &domain.Refund{
		ID:        refund1ID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(600),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now().Add(-2 * time.Second),
		UpdatedAt: time.Now().Add(-2 * time.Second),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref1))

	refund2ID := uuid.New()
	ref2 := &domain.Refund{
		ID:        refund2ID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(600),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now().Add(-1 * time.Second),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref2))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_concur_1",
		Amount:   decimal.NewFromInt(600),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refund1ID.String(),
	}

	var wg sync.WaitGroup
	var err1, err2 error
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err1 = uc.ProcessPendingRefund(ctx, refund1ID)
	}()

	go func() {
		defer wg.Done()
		_, err2 = uc.ProcessPendingRefund(ctx, refund2ID)
	}()

	wg.Wait()

	// Exactly one must succeed, and the other must be rejected for exceeding payment amount!
	successes := 0
	capRejections := 0
	if err1 == nil {
		successes++
	} else if errors.Is(err1, domain.ErrRefundExceedsPayment) {
		capRejections++
	}
	if err2 == nil {
		successes++
	} else if errors.Is(err2, domain.ErrRefundExceedsPayment) {
		capRejections++
	}

	assert.Equal(t, 1, successes, "Exactly one refund should succeed")
	assert.Equal(t, 1, capRejections, "Second refund must be rejected with ErrRefundExceedsPayment")
	assert.Equal(t, 1, provider.refundCalls, "Gateway refund must only be called once")
}

func TestRefundUsecase_ProcessPendingRefund_ConcurrentExecutionsOfSameRefund_LeaseClaim(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test_same"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_same_test",
		Amount:   decimal.NewFromInt(500),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	// First execution claims lease and processes successfully
	res1, err1 := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err1)
	assert.Equal(t, domain.RefundStatusSuccess, res1.Status)

	// Direct execution when already PROCESSING with active lease returns ErrRefundInProgress
	// Simulate an active in-flight lease
	activeRefundID := uuid.New()
	activeRef := &domain.Refund{
		ID:        activeRefundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(200),
		Status:    domain.RefundStatusProcessing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(), // active lease
	}
	require.NoError(t, repos.refundRepo.Create(ctx, activeRef))

	res2, err2 := uc.ProcessPendingRefund(ctx, activeRefundID)
	assert.Error(t, err2)
	assert.True(t, errors.Is(err2, domain.ErrRefundInProgress))
	assert.Nil(t, res2)
}

func TestRefundUsecase_ProcessPendingRefund_PreviouslyUsedGatewayIDNotReused(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	// Existing refund record already used "rfnd_used_1"
	usedGatewayID := "rfnd_used_1"
	existingRef := &domain.Refund{
		ID:              uuid.New(),
		BookingID:       uuid.New(),
		PaymentID:       paymentID,
		GatewayRefundID: &usedGatewayID,
		Amount:          decimal.NewFromInt(400),
		Status:          domain.RefundStatusSuccess,
		CreatedAt:       time.Now().Add(-10 * time.Minute),
		UpdatedAt:       time.Now().Add(-10 * time.Minute),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, existingRef))

	// New refund requests 400
	newRefundID := uuid.New()
	newRef := &domain.Refund{
		ID:        newRefundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(400),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, newRef))

	// Gateway returns the old refund rfnd_used_1
	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: "rfnd_used_1",
			Amount:   decimal.NewFromInt(400),
			Status:   domain.RefundStatusSuccess,
			Receipt:  "some_other_receipt",
		},
	}

	// New refund provider call returns a fresh refund
	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_brand_new_2",
		Amount:   decimal.NewFromInt(400),
		Status:   domain.RefundStatusSuccess,
		Receipt:  newRefundID.String(),
	}

	result, err := uc.ProcessPendingRefund(ctx, newRefundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.Equal(t, "rfnd_brand_new_2", *result.GatewayRefundID, "Must NOT reuse rfnd_used_1")
	assert.Equal(t, 1, provider.refundCalls, "Must call gateway refund API because rfnd_used_1 is already claimed")
}

func TestRefundUsecase_ProcessPendingRefund_AmountOnlyMatchingRejected(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	newRefundID := uuid.New()
	newRef := &domain.Refund{
		ID:        newRefundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, newRef))

	// Gateway has exactly 1 refund of 500, but receipt does NOT match our refund ID
	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: "rfnd_unrelated_500",
			Amount:   decimal.NewFromInt(500),
			Status:   domain.RefundStatusSuccess,
			Receipt:  "unrelated_receipt_uuid",
		},
	}

	// New gateway refund returns new refund
	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_new_genuine",
		Amount:   decimal.NewFromInt(500),
		Status:   domain.RefundStatusSuccess,
		Receipt:  newRefundID.String(),
	}

	result, err := uc.ProcessPendingRefund(ctx, newRefundID)
	require.NoError(t, err)
	assert.Equal(t, "rfnd_new_genuine", *result.GatewayRefundID)
	assert.Equal(t, 1, provider.refundCalls, "Amount-only match must NOT prevent issuing new refund")
}

func TestRefundUsecase_ProcessPendingRefund_ExistingGatewayID_AuthoritativeStatusFetched(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	existingGatewayID := "rfnd_stale_123"
	// Stale local status is PROCESSING
	ref := &domain.Refund{
		ID:              refundID,
		BookingID:       uuid.New(),
		PaymentID:       paymentID,
		GatewayRefundID: &existingGatewayID,
		Amount:          decimal.NewFromInt(500),
		Status:          domain.RefundStatusProcessing,
		CreatedAt:       time.Now().Add(-10 * time.Minute),
		UpdatedAt:       time.Now().Add(-10 * time.Minute), // expired lease
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	// Gateway confirms the refund processed successfully!
	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: existingGatewayID,
			Amount:   decimal.NewFromInt(500),
			Status:   "processed",
			Receipt:  refundID.String(),
		},
	}

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.Equal(t, existingGatewayID, *result.GatewayRefundID)
	assert.NotNil(t, result.ProcessedAt)
	assert.Equal(t, 0, provider.refundCalls, "Should not make a new refund request when existing gateway refund succeeded")
}

func TestRefundUsecase_ProcessPendingRefund_PreflightReconciliation(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, ref)

	// Gateway already has this refund via receipt match!
	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: "rfnd_already_there",
			Amount:   decimal.NewFromInt(500),
			Status:   domain.RefundStatusSuccess,
			Receipt:  refundID.String(),
		},
	}

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.Equal(t, "rfnd_already_there", *result.GatewayRefundID)
	assert.Equal(t, 0, provider.refundCalls, "Should not make refund API call if reconciled preflight")
}

func TestRefundUsecase_ProcessPendingRefund_PostFailureTimeoutRecovery(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, ref)

	// Refund call times out / network error
	provider.refundErr = errors.New("gateway connection timeout")

	// Post-failure reconciliation finds the refund created
	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: "rfnd_recovered_timeout",
			Amount:   decimal.NewFromInt(500),
			Status:   domain.RefundStatusSuccess,
			Receipt:  refundID.String(),
		},
	}

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, result.Status)
	assert.Equal(t, "rfnd_recovered_timeout", *result.GatewayRefundID)
}

func TestRefundUsecase_ProcessPendingRefund_ConfirmedGatewayFailure(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, ref)

	provider.refundErr = errors.New("insufficient balance in merchant account")
	provider.fetchRefunds = nil // no refunds on gateway

	result, err := uc.ProcessPendingRefund(ctx, refundID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrGatewayRefundFailed))
	assert.Nil(t, result)

	// Check DB status is marked FAILED
	saved, findErr := repos.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, findErr)
	assert.Equal(t, domain.RefundStatusFailed, saved.Status)
}

func TestRefundUsecase_ReconcileStuckRefunds_Success(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	ctx := context.Background()

	paymentID := uuid.New()
	txnID := "pay_test123"
	payment := &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1000),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}
	repos.paymentRepo.paymentsByID[paymentID.String()] = payment

	stuckTime := time.Now().Add(-10 * time.Minute)
	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: uuid.New(),
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(500),
		Status:    domain.RefundStatusProcessing,
		CreatedAt: stuckTime,
		UpdatedAt: stuckTime,
	}
	_ = repos.refundRepo.Create(ctx, ref)

	provider.fetchRefunds = []domain.RefundResult{
		{
			RefundID: "rfnd_reconciled_stuck",
			Amount:   decimal.NewFromInt(500),
			Status:   domain.RefundStatusSuccess,
			Receipt:  refundID.String(),
		},
	}

	count, err := uc.ReconcileStuckRefunds(ctx, 5*time.Minute, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	saved, _ := repos.refundRepo.FindByID(ctx, refundID)
	assert.Equal(t, domain.RefundStatusSuccess, saved.Status)
	assert.Equal(t, "rfnd_reconciled_stuck", *saved.GatewayRefundID)
	assert.NotNil(t, saved.ProcessedAt)
}

func TestRefundUsecase_GetRefundByID_And_GetRefundsByBookingID(t *testing.T) {
	uc, repos, _ := setupRefundUsecase(t)
	ctx := context.Background()

	bookingID := uuid.New()
	refundID := uuid.New()
	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: uuid.New(),
		Amount:    decimal.NewFromInt(200),
		Status:    domain.RefundStatusSuccess,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repos.refundRepo.Create(ctx, ref)

	byID, err := uc.GetRefundByID(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, refundID, byID.ID)

	byBooking, err := uc.GetRefundsByBookingID(ctx, bookingID)
	require.NoError(t, err)
	assert.Len(t, byBooking, 1)
	assert.Equal(t, refundID, byBooking[0].ID)
}

func TestRefundUsecase_LedgerIntegration_Success(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	financeUC := &mockFinanceUsecase{}
	uc.SetFinanceUsecase(financeUC)
	ctx := context.Background()

	organizerID := uuid.New()
	eventID := uuid.New()
	bookingID := uuid.New()
	paymentID := uuid.New()
	refundID := uuid.New()

	repos.eventRepo.eventsByID[eventID] = &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
	}

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:          bookingID,
		EventID:     eventID,
		PlatformFee: decimal.Zero,
		Total:       decimal.NewFromInt(500),
		Status:      domain.BookingStatusConfirmed,
	}

	txnID := "pay_test_ledger"
	repos.paymentRepo.paymentsByID[paymentID.String()] = &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(500),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
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
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_ledger_001",
		Amount:   decimal.NewFromInt(500),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	res, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, res.Status)

	// Verify finance journal was recorded
	require.Len(t, financeUC.refundInputs, 1)
	in := financeUC.refundInputs[0]
	assert.Equal(t, refundID, in.RefundID)
	assert.True(t, decimal.NewFromInt(500).Equal(in.RefundAmount))
	assert.Equal(t, paymentID, in.PaymentID)
	assert.Equal(t, bookingID, in.BookingID)
	assert.Equal(t, eventID, in.EventID)
	assert.Equal(t, organizerID, in.OrganizerID)
	assert.False(t, in.IsEventCancellation)
	assert.False(t, in.IsEscrowRefund)
	assert.True(t, in.PlatformFeeRefund.IsZero())
}

func TestRefundUsecase_LedgerIntegration_AtomicRollback(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	financeUC := &mockFinanceUsecase{}
	financeUC.recordRefundWithinTxFn = func(ctx context.Context, txRepos domain.TransactionRepositories, in domain.RecordRefundInput) (*domainfinance.JournalTransaction, error) {
		return nil, errors.New("simulated database ledger failure")
	}
	uc.SetFinanceUsecase(financeUC)
	ctx := context.Background()

	eventID := uuid.New()
	bookingID := uuid.New()
	paymentID := uuid.New()
	refundID := uuid.New()

	repos.eventRepo.eventsByID[eventID] = &domain.Event{
		ID:          eventID,
		OrganizerID: uuid.New(),
	}

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:          bookingID,
		EventID:     eventID,
		Status:      domain.BookingStatusConfirmed,
	}

	txnID := "pay_test_rollback"
	repos.paymentRepo.paymentsByID[paymentID.String()] = &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(400),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}

	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(400),
		Status:    domain.RefundStatusPending,
		Reason:    "CUSTOMER_CANCELLATION",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_rollback_001",
		Amount:   decimal.NewFromInt(400),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	_, err := uc.ProcessPendingRefund(ctx, refundID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated database ledger failure")

	// Atomicity verification: status in repository must NOT be SUCCESS
	// When transaction manager rolled back, the refund status remained PENDING.
	saved, err := repos.refundRepo.FindByID(ctx, refundID)
	require.NoError(t, err)
	assert.NotEqual(t, domain.RefundStatusSuccess, saved.Status)
}

func TestRefundUsecase_LedgerIntegration_EscrowRefund(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	financeUC := &mockFinanceUsecase{}
	uc.SetFinanceUsecase(financeUC)
	ctx := context.Background()

	bookingID := uuid.New()
	paymentID := uuid.New()
	refundID := uuid.New()

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:     bookingID,
		Status: domain.BookingStatusCancelled,
	}

	txnID := "pay_test_escrow"
	repos.paymentRepo.paymentsByID[paymentID.String()] = &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(600),
		Status:        domain.PaymentStatusRefundRequired, // Escrow condition!
		TransactionID: &txnID,
	}

	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(600),
		Status:    domain.RefundStatusPending,
		Reason:    "EVENT_CANCELLATION",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_escrow_001",
		Amount:   decimal.NewFromInt(600),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	res, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, res.Status)

	require.Len(t, financeUC.refundInputs, 1)
	in := financeUC.refundInputs[0]
	assert.True(t, in.IsEscrowRefund, "Must be tagged as escrow refund")
	assert.False(t, in.IsEventCancellation)
}

func TestRefundUsecase_LedgerIntegration_EventCancellation(t *testing.T) {
	uc, repos, provider := setupRefundUsecase(t)
	financeUC := &mockFinanceUsecase{}
	uc.SetFinanceUsecase(financeUC)
	ctx := context.Background()

	organizerID := uuid.New()
	eventID := uuid.New()
	bookingID := uuid.New()
	paymentID := uuid.New()
	refundID := uuid.New()

	repos.eventRepo.eventsByID[eventID] = &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
	}

	repos.bookingRepo.bookingsByID[bookingID] = &domain.Booking{
		ID:          bookingID,
		EventID:     eventID,
		PlatformFee: decimal.NewFromInt(50),
		Total:       decimal.NewFromInt(1050),
		Status:      domain.BookingStatusCancelled,
	}

	txnID := "pay_test_event_cancel"
	repos.paymentRepo.paymentsByID[paymentID.String()] = &domain.Payment{
		ID:            paymentID.String(),
		Amount:        decimal.NewFromInt(1050),
		Status:        domain.PaymentStatusSuccess,
		TransactionID: &txnID,
	}

	ref := &domain.Refund{
		ID:        refundID,
		BookingID: bookingID,
		PaymentID: paymentID,
		Amount:    decimal.NewFromInt(1050),
		Status:    domain.RefundStatusPending,
		Reason:    "EVENT_CANCELLATION",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.refundRepo.Create(ctx, ref))

	provider.refundResult = &domain.RefundResult{
		RefundID: "rfnd_ev_cancel_001",
		Amount:   decimal.NewFromInt(1050),
		Status:   domain.RefundStatusSuccess,
		Receipt:  refundID.String(),
	}

	res, err := uc.ProcessPendingRefund(ctx, refundID)
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSuccess, res.Status)

	require.Len(t, financeUC.refundInputs, 1)
	in := financeUC.refundInputs[0]
	assert.True(t, in.IsEventCancellation, "Must be tagged as event cancellation")
	assert.False(t, in.IsEscrowRefund)
	assert.True(t, decimal.NewFromInt(50).Equal(in.PlatformFeeRefund), "Platform fee portion must match booking platform fee")
}

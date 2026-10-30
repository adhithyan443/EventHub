package checkout

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReservationRepo struct {
	reservation *domain.SeatReservation
	items       []domain.SeatReservationItem
	err         error
}

func (m *mockReservationRepo) Create(ctx context.Context, reservation *domain.SeatReservation) error {
	return nil
}
func (m *mockReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	return nil
}
func (m *mockReservationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.reservation, nil
}
func (m *mockReservationRepo) FindItemsByReservationID(ctx context.Context, id uuid.UUID) ([]domain.SeatReservationItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}
func (m *mockReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockReservationRepo) CompleteIfActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockReservationRepo) ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return true, nil
}
func (m *mockReservationRepo) FindExpiredActive(ctx context.Context, now time.Time, limit int) ([]domain.SeatReservation, error) {
	return nil, nil
}
func (m *mockReservationRepo) CancelIfActive(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.reservation, nil
}
func (m *mockReservationRepo) CountActiveByEventID(ctx context.Context, eventID uuid.UUID, now time.Time) (int64, error) {
	return 0, nil
}

type mockCheckoutRepo struct {
	checkout   *domain.Checkout
	byResID    map[uuid.UUID]*domain.Checkout
	created    []*domain.Checkout
	findErr    error
}

func (m *mockCheckoutRepo) Create(ctx context.Context, checkout *domain.Checkout) error {
	m.created = append(m.created, checkout)
	if m.byResID == nil {
		m.byResID = make(map[uuid.UUID]*domain.Checkout)
	}
	m.byResID[checkout.ReservationID] = checkout
	return nil
}
func (m *mockCheckoutRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Checkout, error) {
	return m.checkout, m.findErr
}
func (m *mockCheckoutRepo) FindByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.Checkout, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if chk, ok := m.byResID[reservationID]; ok {
		return chk, nil
	}
	return nil, domain.ErrCheckoutNotFound
}
func (m *mockCheckoutRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockCheckoutRepo) ExpireActiveByReservationID(ctx context.Context, reservationID uuid.UUID) (bool, error) {
	return true, nil
}

type mockPricingTaxRepo struct {
	config           *domain.PricingTaxConfiguration
	findActiveAtFunc func(t time.Time) (*domain.PricingTaxConfiguration, error)
	err              error
}

func (m *mockPricingTaxRepo) FindActiveAt(ctx context.Context, t time.Time) (*domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.findActiveAtFunc != nil {
		return m.findActiveAtFunc(t)
	}
	return m.config, nil
}
func (m *mockPricingTaxRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.PricingTaxConfiguration, error) {
	return nil, nil
}
func (m *mockPricingTaxRepo) FindLatest(ctx context.Context) (*domain.PricingTaxConfiguration, error) {
	return nil, nil
}
func (m *mockPricingTaxRepo) FindTimeline(ctx context.Context, asOf time.Time) ([]domain.PricingTaxConfiguration, error) {
	return nil, nil
}
func (m *mockPricingTaxRepo) FindHistory(ctx context.Context, page, limit int, asOf time.Time) ([]domain.PricingTaxConfiguration, int64, error) {
	return nil, 0, nil
}
func (m *mockPricingTaxRepo) Create(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	return nil
}
func (m *mockPricingTaxRepo) Update(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	return nil
}
func (m *mockPricingTaxRepo) AcquireAdvisoryLock(ctx context.Context) error {
	return nil
}

type mockTxRepos struct {
	domain.TransactionRepositories
	resRepo        *mockReservationRepo
	checkoutRepo   *mockCheckoutRepo
	pricingTaxRepo *mockPricingTaxRepo
}

func (r *mockTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return r.resRepo
}
func (r *mockTxRepos) CheckoutRepository() domain.CheckoutRepository {
	return r.checkoutRepo
}
func (r *mockTxRepos) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return r.pricingTaxRepo
}

type mockTxManager struct {
	repos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func setupTestCheckoutUsecase(resRepo *mockReservationRepo, checkoutRepo *mockCheckoutRepo, pricingRepo *mockPricingTaxRepo) *CheckoutUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repos := &mockTxRepos{
		resRepo:        resRepo,
		checkoutRepo:   checkoutRepo,
		pricingTaxRepo: pricingRepo,
	}
	txManager := &mockTxManager{repos: repos}
	return NewCheckoutUsecase(txManager, logger)
}

func TestCheckoutUsecase_CreateCheckout(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	resID := uuid.New()
	now := time.Now().UTC()

	activeReservation := &domain.SeatReservation{
		ID:        resID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(10 * time.Minute),
	}

	activePricing := &domain.PricingTaxConfiguration{
		PlatformFeeEnabled: true,
		PlatformFeePercent: decimal.NewFromFloat(4.00), // 4%
		TaxEnabled:         false,
		FreeTicketExempt:   true,
	}

	t.Run("calculates fee for paid tickets on aggregate subtotal", func(t *testing.T) {
		resRepo := &mockReservationRepo{
			reservation: activeReservation,
			items: []domain.SeatReservationItem{
				{Quantity: 2, Price: decimal.NewFromInt(250)}, // 500
				{Quantity: 1, Price: decimal.NewFromInt(500)}, // 500
			},
		}
		checkoutRepo := &mockCheckoutRepo{}
		pricingRepo := &mockPricingTaxRepo{config: activePricing}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		chk, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID,
		})

		require.NoError(t, err)
		assert.Equal(t, "1000.00", chk.Subtotal.StringFixed(2))
		// 4% of 1000 = 40.00
		assert.Equal(t, "40.00", chk.PlatformFee.StringFixed(2))
		assert.Equal(t, "0.00", chk.TaxAmount.StringFixed(2))
		assert.Equal(t, "1040.00", chk.TotalAmount.StringFixed(2))
		assert.Equal(t, domain.CheckoutStatusActive, chk.Status)
	})

	t.Run("free tickets are exempt from platform fee", func(t *testing.T) {
		resRepo := &mockReservationRepo{
			reservation: activeReservation,
			items: []domain.SeatReservationItem{
				{Quantity: 3, Price: decimal.Zero},
			},
		}
		checkoutRepo := &mockCheckoutRepo{}
		pricingRepo := &mockPricingTaxRepo{config: activePricing}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		chk, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID,
		})

		require.NoError(t, err)
		assert.True(t, chk.Subtotal.IsZero())
		assert.True(t, chk.PlatformFee.IsZero())
		assert.True(t, chk.TotalAmount.IsZero())
	})

	t.Run("mixed ticket booking calculates fee only on paid items", func(t *testing.T) {
		resRepo := &mockReservationRepo{
			reservation: activeReservation,
			items: []domain.SeatReservationItem{
				{Quantity: 2, Price: decimal.Zero},            // Free
				{Quantity: 1, Price: decimal.NewFromInt(500)}, // Paid: 500
			},
		}
		checkoutRepo := &mockCheckoutRepo{}
		pricingRepo := &mockPricingTaxRepo{config: activePricing}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		chk, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID,
		})

		require.NoError(t, err)
		assert.Equal(t, "500.00", chk.Subtotal.StringFixed(2))
		// 4% of 500 = 20.00
		assert.Equal(t, "20.00", chk.PlatformFee.StringFixed(2))
		assert.Equal(t, "520.00", chk.TotalAmount.StringFixed(2))
	})

	t.Run("fails closed when pricing configuration is missing", func(t *testing.T) {
		resRepo := &mockReservationRepo{
			reservation: activeReservation,
			items: []domain.SeatReservationItem{
				{Quantity: 1, Price: decimal.NewFromInt(500)},
			},
		}
		checkoutRepo := &mockCheckoutRepo{}
		// Simulates error or missing config
		pricingRepo := &mockPricingTaxRepo{
			err: domain.ErrPricingConfigurationUnavailable,
		}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		_, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID,
		})

		require.ErrorIs(t, err, domain.ErrPricingConfigurationUnavailable)
	})

	t.Run("returns existing active checkout snapshot without recomputing", func(t *testing.T) {
		existingCheckout := &domain.Checkout{
			ID:            uuid.New(),
			ReservationID: resID,
			UserID:        userID,
			Subtotal:      decimal.NewFromInt(100),
			PlatformFee:   decimal.NewFromInt(5),
			TotalAmount:   decimal.NewFromInt(105),
			Status:        domain.CheckoutStatusActive,
			ExpiresAt:     now.Add(5 * time.Minute),
		}

		resRepo := &mockReservationRepo{
			reservation: activeReservation,
		}
		checkoutRepo := &mockCheckoutRepo{
			byResID: map[uuid.UUID]*domain.Checkout{
				resID: existingCheckout,
			},
		}
		pricingRepo := &mockPricingTaxRepo{
			// Even if config has different rate, existing snapshot must be returned verbatim!
			config: &domain.PricingTaxConfiguration{
				PlatformFeePercent: decimal.NewFromInt(50),
			},
		}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		chk, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID,
		})

		require.NoError(t, err)
		assert.Equal(t, existingCheckout.ID, chk.ID)
		assert.Equal(t, "5.00", chk.PlatformFee.StringFixed(2))
		assert.Equal(t, "105.00", chk.TotalAmount.StringFixed(2))
	})

	t.Run("checkout selects active pricing at its creation timestamp", func(t *testing.T) {
		boundaryTime := time.Now().Add(10 * time.Minute)
		epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

		v2 := &domain.PricingTaxConfiguration{
			ID:                 uuid.New(),
			Version:            2,
			Status:             domain.PricingTaxStatusActive,
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(4.99),
			FreeTicketExempt:   true,
			EffectiveFrom:      epoch,
			EffectiveTo:        &boundaryTime,
		}

		v3 := &domain.PricingTaxConfiguration{
			ID:                 uuid.New(),
			Version:            3,
			Status:             domain.PricingTaxStatusScheduled,
			PlatformFeeEnabled: true,
			PlatformFeePercent: decimal.NewFromFloat(7.00),
			FreeTicketExempt:   true,
			EffectiveFrom:      boundaryTime,
			EffectiveTo:        nil,
		}

		resolver := func(t time.Time) (*domain.PricingTaxConfiguration, error) {
			if t.Before(boundaryTime) {
				cfg := *v2
				cfg.Status = cfg.ResolveStatus(t)
				return &cfg, nil
			}
			cfg := *v3
			cfg.Status = cfg.ResolveStatus(t)
			return &cfg, nil
		}

		// Reservation 1: evaluated at current time (before boundaryTime)
		resID1 := uuid.New()
		res1 := &domain.SeatReservation{
			ID:        resID1,
			UserID:    userID,
			Status:    domain.ReservationStatusActive,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		items1 := []domain.SeatReservationItem{
			{ID: uuid.New(), ReservationID: resID1, Price: decimal.NewFromInt(1000), Quantity: 1},
		}

		resRepo := &mockReservationRepo{
			reservation: res1,
			items:       items1,
		}
		checkoutRepo := &mockCheckoutRepo{}
		pricingRepo := &mockPricingTaxRepo{
			findActiveAtFunc: resolver,
		}

		uc := setupTestCheckoutUsecase(resRepo, checkoutRepo, pricingRepo)

		// Create checkout 1: fee at 4.99% on 1000 = 49.90
		chk1, err := uc.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID1,
		})
		require.NoError(t, err)
		assert.Equal(t, "49.90", chk1.PlatformFee.StringFixed(2))
		assert.Equal(t, "1049.90", chk1.TotalAmount.StringFixed(2))

		// Reservation 2: simulated after boundaryTime
		resID2 := uuid.New()
		res2 := &domain.SeatReservation{
			ID:        resID2,
			UserID:    userID,
			Status:    domain.ReservationStatusActive,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		items2 := []domain.SeatReservationItem{
			{ID: uuid.New(), ReservationID: resID2, Price: decimal.NewFromInt(1000), Quantity: 1},
		}

		// When resolver is invoked after boundaryTime, v3 applies (7.00%)
		futureResolver := func(t time.Time) (*domain.PricingTaxConfiguration, error) {
			cfg := *v3
			cfg.Status = cfg.ResolveStatus(boundaryTime.Add(1 * time.Second))
			return &cfg, nil
		}

		resRepo2 := &mockReservationRepo{
			reservation: res2,
			items:       items2,
		}
		checkoutRepo2 := &mockCheckoutRepo{}
		pricingRepo2 := &mockPricingTaxRepo{
			findActiveAtFunc: futureResolver,
		}

		uc2 := setupTestCheckoutUsecase(resRepo2, checkoutRepo2, pricingRepo2)
		chk2, err := uc2.CreateCheckout(ctx, CreateCheckoutRequest{
			UserID:        userID,
			ReservationID: resID2,
		})
		require.NoError(t, err)
		assert.Equal(t, "70.00", chk2.PlatformFee.StringFixed(2))
		assert.Equal(t, "1070.00", chk2.TotalAmount.StringFixed(2))
	})
}

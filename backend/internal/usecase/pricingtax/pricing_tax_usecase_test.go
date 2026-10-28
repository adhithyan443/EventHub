package pricingtax

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

type mockPricingTaxRepo struct {
	latest   *domain.PricingTaxConfiguration
	active   *domain.PricingTaxConfiguration
	timeline []domain.PricingTaxConfiguration
	history  []domain.PricingTaxConfiguration
	total    int64
	byID     map[uuid.UUID]*domain.PricingTaxConfiguration
	updated  []*domain.PricingTaxConfiguration
	created  []*domain.PricingTaxConfiguration
	lockErr  error
	err      error
}

func (m *mockPricingTaxRepo) FindActiveAt(ctx context.Context, t time.Time) (*domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.active, nil
}

func (m *mockPricingTaxRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	if cfg, ok := m.byID[id]; ok {
		return cfg, nil
	}
	return nil, domain.ErrScheduledConfigurationNotFound
}

func (m *mockPricingTaxRepo) FindLatest(ctx context.Context) (*domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.latest, nil
}

func (m *mockPricingTaxRepo) FindTimeline(ctx context.Context) ([]domain.PricingTaxConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}
	res := make([]domain.PricingTaxConfiguration, len(m.timeline))
	copy(res, m.timeline)
	return res, nil
}

func (m *mockPricingTaxRepo) FindHistory(ctx context.Context, page, limit int) ([]domain.PricingTaxConfiguration, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.history, m.total, nil
}

func (m *mockPricingTaxRepo) Create(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	m.created = append(m.created, config)
	m.timeline = append(m.timeline, *config)
	m.latest = config
	return nil
}

func (m *mockPricingTaxRepo) Update(ctx context.Context, config *domain.PricingTaxConfiguration) error {
	m.updated = append(m.updated, config)
	for i := range m.timeline {
		if m.timeline[i].ID == config.ID {
			m.timeline[i] = *config
			break
		}
	}
	if m.byID != nil {
		m.byID[config.ID] = config
	}
	return nil
}

func (m *mockPricingTaxRepo) AcquireAdvisoryLock(ctx context.Context) error {
	return m.lockErr
}

type mockTxManager struct {
	repo *mockPricingTaxRepo
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	repos := &mockTxRepos{repo: m.repo}
	return fn(repos)
}

type mockTxRepos struct {
	domain.TransactionRepositories
	repo *mockPricingTaxRepo
}

func (r *mockTxRepos) PricingTaxConfigurationRepository() domain.PricingTaxConfigurationRepository {
	return r.repo
}

func setupTestUsecase(repo *mockPricingTaxRepo) *PricingTaxUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := &mockTxManager{repo: repo}
	return NewPricingTaxUsecase(txManager, logger)
}

func TestPricingTaxUsecase_ValidationRules(t *testing.T) {
	repo := &mockPricingTaxRepo{}
	uc := setupTestUsecase(repo)
	ctx := context.Background()

	t.Run("negative rate rejected", func(t *testing.T) {
		_, err := uc.PublishImmediate(ctx, PublishRequest{
			PlatformFeePercent: decimal.NewFromInt(-1),
			ChangeReason:       "Invalid rate",
		})
		require.ErrorIs(t, err, domain.ErrInvalidPricingConfiguration)
	})

	t.Run("rate over 100 rejected", func(t *testing.T) {
		_, err := uc.PublishImmediate(ctx, PublishRequest{
			PlatformFeePercent: decimal.NewFromInt(105),
			ChangeReason:       "Invalid rate",
		})
		require.ErrorIs(t, err, domain.ErrInvalidPricingConfiguration)
	})

	t.Run("empty reason rejected", func(t *testing.T) {
		_, err := uc.PublishImmediate(ctx, PublishRequest{
			PlatformFeePercent: decimal.NewFromInt(5),
			ChangeReason:       "   ",
		})
		require.ErrorIs(t, err, domain.ErrInvalidPricingConfiguration)
	})

	t.Run("enabling tax without validation reference rejected", func(t *testing.T) {
		_, err := uc.PublishImmediate(ctx, PublishRequest{
			PlatformFeePercent: decimal.NewFromInt(5),
			TaxEnabled:         true,
			TaxPercent:         decimal.NewFromInt(18),
			TaxBasis:           domain.TaxBasisPlatformFeeOnly,
			TaxRulesValidated:  false, // Not validated
			ChangeReason:       "Enable tax",
		})
		require.ErrorIs(t, err, domain.ErrTaxBasisUnvalidated)

		_, err = uc.PublishImmediate(ctx, PublishRequest{
			PlatformFeePercent:       decimal.NewFromInt(5),
			TaxEnabled:               true,
			TaxPercent:               decimal.NewFromInt(18),
			TaxBasis:                 domain.TaxBasisPlatformFeeOnly,
			TaxRulesValidated:        true,
			TaxLegalSignoffReference: nil,
			ChangeReason:             "Enable tax",
		})
		require.ErrorIs(t, err, domain.ErrTaxBasisUnvalidated)
	})

	t.Run("scheduling in the past rejected", func(t *testing.T) {
		_, err := uc.ScheduleFuture(ctx, ScheduleRequest{
			EffectiveFrom: time.Now().UTC().Add(-1 * time.Hour),
			ChangeReason:  "Past date",
		})
		require.ErrorIs(t, err, domain.ErrInvalidEffectiveDate)
	})
}

func TestPricingTaxUsecase_ConcurrentConflict(t *testing.T) {
	repo := &mockPricingTaxRepo{
		latest: &domain.PricingTaxConfiguration{Version: 2},
	}
	uc := setupTestUsecase(repo)
	ctx := context.Background()

	// Admin submits expecting base version 1 while DB has version 2
	_, err := uc.PublishImmediate(ctx, PublishRequest{
		ExpectedBaseVersion: 1,
		PlatformFeeEnabled:  true,
		PlatformFeePercent:  decimal.NewFromInt(5),
		ChangeReason:        "Concurrent update",
	})

	require.ErrorIs(t, err, domain.ErrConcurrentConfigurationModification)
}

func TestPricingTaxUsecase_PreserveFutureSchedulesOnImmediatePublish(t *testing.T) {
	now := time.Now().UTC()
	futureTime := now.Add(7 * 24 * time.Hour)
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	v1 := domain.PricingTaxConfiguration{
		ID:            uuid.New(),
		Version:       1,
		Status:        domain.PricingTaxStatusActive,
		EffectiveFrom: epoch,
		EffectiveTo:   &futureTime,
	}
	v2 := domain.PricingTaxConfiguration{
		ID:            uuid.New(),
		Version:       2,
		Status:        domain.PricingTaxStatusScheduled,
		EffectiveFrom: futureTime,
		EffectiveTo:   nil,
	}

	repo := &mockPricingTaxRepo{
		latest:   &v2,
		timeline: []domain.PricingTaxConfiguration{v1, v2},
	}
	uc := setupTestUsecase(repo)
	ctx := context.Background()

	// Publish immediate without replace_future_schedules flag (default: preserve)
	res, err := uc.PublishImmediate(ctx, PublishRequest{
		ExpectedBaseVersion:    2,
		PlatformFeeEnabled:     true,
		PlatformFeePercent:     decimal.NewFromInt(5),
		ReplaceFutureSchedules: false, // PRESERVE
		ChangeReason:           "Immediate adjustment",
		AdminUserID:            uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, 3, res.Version)
	assert.Equal(t, domain.PricingTaxStatusActive, res.Status)
	// Must be bounded by the future schedule's start time!
	require.NotNil(t, res.EffectiveTo)
	assert.Equal(t, futureTime, *res.EffectiveTo)
}

func TestPricingTaxUsecase_CancelScheduledRestoresPredecessor(t *testing.T) {
	now := time.Now().UTC()
	futureTime := now.Add(7 * 24 * time.Hour)
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	v1ID := uuid.New()
	v2ID := uuid.New()

	v1 := domain.PricingTaxConfiguration{
		ID:            v1ID,
		Version:       1,
		Status:        domain.PricingTaxStatusActive,
		EffectiveFrom: epoch,
		EffectiveTo:   &futureTime,
	}
	v2 := domain.PricingTaxConfiguration{
		ID:            v2ID,
		Version:       2,
		Status:        domain.PricingTaxStatusScheduled,
		EffectiveFrom: futureTime,
		EffectiveTo:   nil,
	}

	repo := &mockPricingTaxRepo{
		latest:   &v2,
		timeline: []domain.PricingTaxConfiguration{v1, v2},
		byID: map[uuid.UUID]*domain.PricingTaxConfiguration{
			v1ID: &v1,
			v2ID: &v2,
		},
	}
	uc := setupTestUsecase(repo)
	ctx := context.Background()

	err := uc.CancelScheduled(ctx, v2ID, uuid.New(), "Cancelled by admin", 2)
	require.NoError(t, err)

	// Predecessor v1 must be restored to target v2's EffectiveTo (which is nil)
	assert.Nil(t, repo.byID[v1ID].EffectiveTo)
	assert.Equal(t, domain.PricingTaxStatusCancelled, repo.byID[v2ID].Status)
}

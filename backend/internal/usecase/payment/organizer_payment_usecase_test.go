package payment_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerPaymentRepo struct {
	domain.PaymentRepository
	lastFilter          *domain.OrganizerPaymentFilter
	listResult          *domain.OrganizerPaymentListResult
	summaryResult       *domain.OrganizerEventPaymentSummaryResult
	overviewResult      *domain.OrganizerPaymentOverview
	listErr             error
	summaryErr          error
	overviewErr         error
}

func (m *mockOrganizerPaymentRepo) FindOrganizerPayments(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentListResult, error) {
	m.lastFilter = &filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockOrganizerPaymentRepo) GetOrganizerEventPaymentSummaries(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerEventPaymentSummaryResult, error) {
	m.lastFilter = &filter
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summaryResult, nil
}

func (m *mockOrganizerPaymentRepo) GetOrganizerPaymentOverview(
	ctx context.Context,
	filter domain.OrganizerPaymentFilter,
) (*domain.OrganizerPaymentOverview, error) {
	m.lastFilter = &filter
	if m.overviewErr != nil {
		return nil, m.overviewErr
	}
	return m.overviewResult, nil
}

type mockOrganizerRepoForPaymentTest struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
	err        error
}

func (m *mockOrganizerRepoForPaymentTest) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	if org, ok := m.organizers[userID]; ok {
		return org, nil
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockEventRepoForPaymentTest struct {
	domain.EventRepository
	events map[uuid.UUID]*domain.Event
	err    error
}

func (m *mockEventRepoForPaymentTest) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ev, ok := m.events[id]; ok {
		return ev, nil
	}
	return nil, domain.ErrEventNotFound
}

func TestNewOrganizerPaymentUsecase(t *testing.T) {
	t.Run("nil payment repository returns ErrNilPaymentRepository", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{}
		uc, err := payment.NewOrganizerPaymentUsecase(nil, orgRepo, nil, slog.Default())
		assert.Nil(t, uc)
		assert.ErrorIs(t, err, payment.ErrNilPaymentRepository)
	})

	t.Run("nil organizer repository returns ErrNilOrganizerRepository", func(t *testing.T) {
		payRepo := &mockOrganizerPaymentRepo{}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, nil, nil, slog.Default())
		assert.Nil(t, uc)
		assert.ErrorIs(t, err, payment.ErrNilOrganizerRepository)
	})

	t.Run("nil logger defaults to slog.Default without error", func(t *testing.T) {
		payRepo := &mockOrganizerPaymentRepo{}
		orgRepo := &mockOrganizerRepoForPaymentTest{}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})

	t.Run("valid parameters construct successfully", func(t *testing.T) {
		payRepo := &mockOrganizerPaymentRepo{}
		orgRepo := &mockOrganizerRepoForPaymentTest{}
		eventRepo := &mockEventRepoForPaymentTest{}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, eventRepo, slog.Default())
		require.NoError(t, err)
		assert.NotNil(t, uc)
	})
}

func TestOrganizerPaymentUsecase_Authorization(t *testing.T) {
	ctx := context.Background()
	payRepo := &mockOrganizerPaymentRepo{
		overviewResult: &domain.OrganizerPaymentOverview{},
	}
	userID := uuid.New()
	orgID := uuid.New()

	t.Run("missing authenticated user ID returns ErrInvalidUserID", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		_, err = uc.GetOrganizerPaymentOverview(ctx, uuid.Nil, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, payment.ErrInvalidUserID)
	})

	t.Run("non-organizer user returns ErrUnauthorizedOrganizer", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{
			organizers: make(map[uuid.UUID]*domain.Organizer),
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		_, err = uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, payment.ErrUnauthorizedOrganizer)
	})

	t.Run("inactive organizer status returns ErrUnauthorizedOrganizer", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				userID: {
					ID:     orgID,
					UserID: userID,
					Status: "PENDING",
				},
			},
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		_, err = uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, payment.ErrUnauthorizedOrganizer)
	})

	t.Run("approved organizer status is accepted", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				userID: {
					ID:     orgID,
					UserID: userID,
					Status: "APPROVED",
				},
			},
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		res, err := uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		require.NoError(t, err)
		assert.NotNil(t, res)
		require.NotNil(t, payRepo.lastFilter)
		assert.Equal(t, orgID, payRepo.lastFilter.OrganizerID)
	})

	t.Run("active organizer status is accepted", func(t *testing.T) {
		orgRepo := &mockOrganizerRepoForPaymentTest{
			organizers: map[uuid.UUID]*domain.Organizer{
				userID: {
					ID:     orgID,
					UserID: userID,
					Status: "ACTIVE",
				},
			},
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		res, err := uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		require.NoError(t, err)
		assert.NotNil(t, res)
		require.NotNil(t, payRepo.lastFilter)
		assert.Equal(t, orgID, payRepo.lastFilter.OrganizerID)
	})
}

func TestOrganizerPaymentUsecase_ValidationAndScoping(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()
	foreignOrgID := uuid.New()

	orgRepo := &mockOrganizerRepoForPaymentTest{
		organizers: map[uuid.UUID]*domain.Organizer{
			userID: {
				ID:     orgID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
	}

	eventID1 := uuid.New()
	foreignEventID := uuid.New()

	eventRepo := &mockEventRepoForPaymentTest{
		events: map[uuid.UUID]*domain.Event{
			eventID1: {
				ID:          eventID1,
				OrganizerID: orgID,
				Title:       "My Event",
			},
			foreignEventID: {
				ID:          foreignEventID,
				OrganizerID: foreignOrgID,
				Title:       "Foreign Event",
			},
		},
	}

	payRepo := &mockOrganizerPaymentRepo{
		listResult:     &domain.OrganizerPaymentListResult{},
		summaryResult:  &domain.OrganizerEventPaymentSummaryResult{},
		overviewResult: &domain.OrganizerPaymentOverview{},
	}

	uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, eventRepo, slog.Default())
	require.NoError(t, err)

	t.Run("never accepts organizer ID from request - enforces authenticated organizer ID", func(t *testing.T) {
		spoofedOrgID := uuid.New()
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			OrganizerID: spoofedOrgID,
		})
		require.NoError(t, err)
		require.NotNil(t, payRepo.lastFilter)
		assert.Equal(t, orgID, payRepo.lastFilter.OrganizerID, "Must override with authenticated organizer ID")
	})

	t.Run("normalizes pagination bounds", func(t *testing.T) {
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			Page:  0,
			Limit: -1,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, payRepo.lastFilter.Page)
		assert.Equal(t, 10, payRepo.lastFilter.Limit)

		_, err = uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			Page:  200_000,
			Limit: 500,
		})
		require.NoError(t, err)
		assert.Equal(t, 100_000, payRepo.lastFilter.Page)
		assert.Equal(t, 100, payRepo.lastFilter.Limit)
	})

	t.Run("validates payment status filter", func(t *testing.T) {
		validStatuses := []string{"SUCCESS", "PENDING", "FAILED", "REFUND_REQUIRED", "ALL", ""}
		for _, s := range validStatuses {
			_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
				Status: s,
			})
			assert.NoError(t, err, "Status '%s' should be valid", s)
		}

		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			Status: "INVALID_STATUS",
		})
		assert.ErrorIs(t, err, payment.ErrInvalidPaymentStatus)
	})

	t.Run("validates allocation status filter", func(t *testing.T) {
		validStatuses := []string{"ACCOUNTED", "UNALLOCATED", "NOT_APPLICABLE", "ALL", ""}
		for _, s := range validStatuses {
			_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
				AllocationStatus: s,
			})
			assert.NoError(t, err, "AllocationStatus '%s' should be valid", s)
		}

		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			AllocationStatus: "SOME_RANDOM_STATUS",
		})
		assert.ErrorIs(t, err, payment.ErrInvalidAllocationStatus)
	})

	t.Run("validates date range", func(t *testing.T) {
		now := time.Now().UTC()
		past := now.Add(-24 * time.Hour)
		future := now.Add(24 * time.Hour)

		// Valid range
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			From: &past,
			To:   &future,
		})
		assert.NoError(t, err)

		// Invalid: From after To
		_, err = uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			From: &future,
			To:   &past,
		})
		assert.ErrorIs(t, err, payment.ErrInvalidDateRange)
	})

	t.Run("event filtering: owned event succeeds", func(t *testing.T) {
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			EventID: &eventID1,
		})
		assert.NoError(t, err)
		assert.Equal(t, &eventID1, payRepo.lastFilter.EventID)
	})

	t.Run("event filtering: nil UUID event ID rejected", func(t *testing.T) {
		nilUUID := uuid.Nil
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			EventID: &nilUUID,
		})
		assert.ErrorIs(t, err, payment.ErrInvalidEventID)
	})

	t.Run("event filtering: foreign event returns ErrUnauthorizedEvent", func(t *testing.T) {
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			EventID: &foreignEventID,
		})
		assert.ErrorIs(t, err, payment.ErrUnauthorizedEvent)
	})

	t.Run("event filtering: non-existent event returns ErrEventNotFound", func(t *testing.T) {
		missingEventID := uuid.New()
		_, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{
			EventID: &missingEventID,
		})
		assert.ErrorIs(t, err, payment.ErrEventNotFound)
	})
}

func TestOrganizerPaymentUsecase_OperationsAndErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()

	orgRepo := &mockOrganizerRepoForPaymentTest{
		organizers: map[uuid.UUID]*domain.Organizer{
			userID: {
				ID:     orgID,
				UserID: userID,
				Status: "ACTIVE",
			},
		},
	}

	t.Run("GetOrganizerPaymentOverview succeeds and propagates repo error", func(t *testing.T) {
		expectedOverview := &domain.OrganizerPaymentOverview{
			GrossCapturedAmount: decimal.NewFromInt(1500),
			TotalTicketsSold:    5,
			TotalEvents:         2,
		}
		payRepo := &mockOrganizerPaymentRepo{
			overviewResult: expectedOverview,
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		res, err := uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		require.NoError(t, err)
		assert.Equal(t, expectedOverview, res)

		// Repo error propagation
		expectedErr := errors.New("database connection down")
		payRepo.overviewErr = expectedErr
		_, err = uc.GetOrganizerPaymentOverview(ctx, userID, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("GetOrganizerEventPaymentSummaries succeeds and propagates repo error", func(t *testing.T) {
		expectedResult := &domain.OrganizerEventPaymentSummaryResult{
			Summaries:    []domain.OrganizerEventPaymentSummary{},
			TotalRecords: 0,
		}
		payRepo := &mockOrganizerPaymentRepo{
			summaryResult: expectedResult,
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		res, err := uc.GetOrganizerEventPaymentSummaries(ctx, userID, domain.OrganizerPaymentFilter{})
		require.NoError(t, err)
		assert.Equal(t, expectedResult, res)

		// Repo error propagation
		expectedErr := errors.New("database query timed out")
		payRepo.summaryErr = expectedErr
		_, err = uc.GetOrganizerEventPaymentSummaries(ctx, userID, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("FindOrganizerPayments succeeds and propagates repo error", func(t *testing.T) {
		expectedResult := &domain.OrganizerPaymentListResult{
			Payments:     []domain.OrganizerPaymentListItem{},
			TotalRecords: 0,
		}
		payRepo := &mockOrganizerPaymentRepo{
			listResult: expectedResult,
		}
		uc, err := payment.NewOrganizerPaymentUsecase(payRepo, orgRepo, nil, slog.Default())
		require.NoError(t, err)

		res, err := uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{})
		require.NoError(t, err)
		assert.Equal(t, expectedResult, res)

		// Repo error propagation
		expectedErr := errors.New("scan failure")
		payRepo.listErr = expectedErr
		_, err = uc.FindOrganizerPayments(ctx, userID, domain.OrganizerPaymentFilter{})
		assert.ErrorIs(t, err, expectedErr)
	})
}

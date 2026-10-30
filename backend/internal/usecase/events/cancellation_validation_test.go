package events

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestValidateCancellationSettings(t *testing.T) {
	tests := []struct {
		name             string
		allowed          bool
		deadlineHours    int
		refundPolicy     string
		refundPercentage int
		wantPolicy       string
		wantErr          error
	}{
		// Valid cases: FULL refund (percentage = 100)
		{
			name:             "valid FULL policy exactly 100 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 100,
			wantPolicy:       domain.RefundPolicyFull,
			wantErr:          nil,
		},
		{
			name:             "valid FULL policy lowercase normalized",
			allowed:          true,
			deadlineHours:    48,
			refundPolicy:     "full",
			refundPercentage: 100,
			wantPolicy:       domain.RefundPolicyFull,
			wantErr:          nil,
		},
		{
			name:             "valid FULL policy with whitespace trimmed",
			allowed:          true,
			deadlineHours:    1,
			refundPolicy:     "  FULL  ",
			refundPercentage: 100,
			wantPolicy:       domain.RefundPolicyFull,
			wantErr:          nil,
		},

		// Valid cases: PARTIAL refund (0 < percentage < 100)
		{
			name:             "valid PARTIAL policy at mid-range 50 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 50,
			wantPolicy:       domain.RefundPolicyPartial,
			wantErr:          nil,
		},
		{
			name:             "valid PARTIAL policy lowercase at lower boundary 1 percent",
			allowed:          true,
			deadlineHours:    12,
			refundPolicy:     "partial",
			refundPercentage: 1,
			wantPolicy:       domain.RefundPolicyPartial,
			wantErr:          nil,
		},
		{
			name:             "valid PARTIAL policy at upper boundary 99 percent",
			allowed:          true,
			deadlineHours:    72,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 99,
			wantPolicy:       domain.RefundPolicyPartial,
			wantErr:          nil,
		},

		// Valid cases: NO_REFUND (percentage = 0)
		{
			name:             "valid NO_REFUND policy with 0 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},
		{
			name:             "valid NO_REFUND policy lowercase with 0 percent",
			allowed:          true,
			deadlineHours:    1,
			refundPolicy:     "no_refund",
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},
		{
			name:             "valid NO_REFUND policy with whitespace trimmed",
			allowed:          true,
			deadlineHours:    12,
			refundPolicy:     "  NO_REFUND  ",
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},

		// Valid cases: Cancellation disabled (allowed = false)
		{
			name:             "valid disabled cancellation with empty policy",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     "",
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},
		{
			name:             "valid disabled cancellation with whitespace policy",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     "   ",
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},
		{
			name:             "valid disabled cancellation with NO_REFUND policy",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},
		{
			name:             "valid disabled cancellation with lowercase no_refund",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     "no_refund",
			refundPercentage: 0,
			wantPolicy:       domain.RefundPolicyNoRefund,
			wantErr:          nil,
		},

		// Invalid cases: FULL violations (must be exactly 100)
		{
			name:             "invalid FULL with 0 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid FULL with 99 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 99,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid FULL with 50 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 50,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid FULL with 101 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 101,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid FULL with negative percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: -10,
			wantErr:          ErrInvalidCancellation,
		},

		// Invalid cases: PARTIAL violations (must be 0 < percentage < 100)
		{
			name:             "invalid PARTIAL with 0 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid PARTIAL with 100 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid PARTIAL with negative percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: -1,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid PARTIAL with over 100 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 105,
			wantErr:          ErrInvalidCancellation,
		},

		// Invalid cases: NO_REFUND violations (must be 0)
		{
			name:             "invalid NO_REFUND with 50 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 50,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid NO_REFUND with 100 percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid NO_REFUND with negative percent",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: -5,
			wantErr:          ErrInvalidCancellation,
		},

		// Invalid cases: Deadline hours
		{
			name:             "invalid zero deadline hours when allowed",
			allowed:          true,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid negative deadline hours when allowed",
			allowed:          true,
			deadlineHours:    -24,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},

		// Invalid cases: Unsupported policy strings
		{
			name:             "invalid empty policy string when allowed",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     "",
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid whitespace policy string when allowed",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     "   ",
			refundPercentage: 100,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid unknown policy name when allowed",
			allowed:          true,
			deadlineHours:    24,
			refundPolicy:     "UNKNOWN_POLICY",
			refundPercentage: 50,
			wantErr:          ErrInvalidCancellation,
		},

		// Invalid cases: Disabled cancellation violations
		{
			name:             "invalid disabled cancellation with non-zero deadline",
			allowed:          false,
			deadlineHours:    24,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid disabled cancellation with non-zero percentage",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: 50,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid disabled cancellation with negative percentage",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyNoRefund,
			refundPercentage: -10,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid disabled cancellation with contradictory FULL policy",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyFull,
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid disabled cancellation with contradictory PARTIAL policy",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     domain.RefundPolicyPartial,
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
		{
			name:             "invalid disabled cancellation with arbitrary policy name",
			allowed:          false,
			deadlineHours:    0,
			refundPolicy:     "RANDOM",
			refundPercentage: 0,
			wantErr:          ErrInvalidCancellation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPolicy, err := validateCancellationSettings(
				tc.allowed,
				tc.deadlineHours,
				tc.refundPolicy,
				tc.refundPercentage,
			)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, gotPolicy)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPolicy, gotPolicy)
			}
		})
	}
}

func buildValidCreateEventInput() CreateEventInput {
	futureDate := time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour)
	startTime := futureDate.Add(10 * time.Hour)
	endTime := futureDate.Add(12 * time.Hour)

	return CreateEventInput{
		UserID:              uuid.New(),
		CategoryID:          uuid.New(),
		Title:               "Launch Party",
		Description:         "Annual community launch event",
		BannerContentType:   "image/png",
		BannerReader:        strings.NewReader("fake-banner-content"),
		BannerSize:          1024,
		Language:            "English",
		AgeRestriction:      18,
		Visibility:          "PUBLIC",
		EventType:           domain.EventTypeOnline,
		OnlineURL:           "https://meet.example.com/event",
		EventDate:           futureDate,
		StartTime:           startTime,
		EndTime:             endTime,
		SeatLayoutType:      SeatLayoutTypeGeneral,
		BookingLimitPerUser: 5,
		TicketTypes: []TicketTypeInput{
			{
				Name:        "General Pass",
				Price:       decimal.RequireFromString("150.00"),
				Capacity:    100,
				Description: "Standard ticket",
			},
		},
		CancellationAllowed:       true,
		CancellationDeadlineHours: 24,
		RefundPolicy:              domain.RefundPolicyFull,
		RefundPercentage:          100,
	}
}

func TestValidateCreateEventInput_CancellationSettings(t *testing.T) {
	t.Run("valid combinations", func(t *testing.T) {
		validConfigs := []struct {
			name       string
			allowed    bool
			deadline   int
			policy     string
			percentage int
		}{
			{"FULL policy with 100%", true, 24, "FULL", 100},
			{"PARTIAL policy with 50%", true, 12, "PARTIAL", 50},
			{"PARTIAL policy with 1%", true, 1, "partial", 1},
			{"PARTIAL policy with 99%", true, 48, "PARTIAL", 99},
			{"NO_REFUND policy with 0%", true, 24, "NO_REFUND", 0},
			{"Disabled cancellation", false, 0, "", 0},
			{"Disabled cancellation with NO_REFUND", false, 0, "NO_REFUND", 0},
		}

		for _, cfg := range validConfigs {
			t.Run(cfg.name, func(t *testing.T) {
				input := buildValidCreateEventInput()
				input.CancellationAllowed = cfg.allowed
				input.CancellationDeadlineHours = cfg.deadline
				input.RefundPolicy = cfg.policy
				input.RefundPercentage = cfg.percentage

				err := validateCreateEventInput(input)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("invalid combinations return ErrInvalidCancellation", func(t *testing.T) {
		invalidConfigs := []struct {
			name       string
			allowed    bool
			deadline   int
			policy     string
			percentage int
		}{
			{"FULL with 90%", true, 24, "FULL", 90},
			{"FULL with 0%", true, 24, "FULL", 0},
			{"PARTIAL with 0%", true, 24, "PARTIAL", 0},
			{"PARTIAL with 100%", true, 24, "PARTIAL", 100},
			{"NO_REFUND with 50%", true, 24, "NO_REFUND", 50},
			{"Allowed with 0 deadline", true, 0, "FULL", 100},
			{"Allowed with empty policy", true, 24, "", 100},
			{"Allowed with unsupported policy", true, 24, "RANDOM", 50},
			{"Disabled with non-zero deadline", false, 24, "", 0},
			{"Disabled with non-zero percentage", false, 0, "", 50},
			{"Disabled with contradictory policy FULL", false, 0, "FULL", 0},
			{"Disabled with contradictory policy PARTIAL", false, 0, "PARTIAL", 0},
		}

		for _, cfg := range invalidConfigs {
			t.Run(cfg.name, func(t *testing.T) {
				input := buildValidCreateEventInput()
				input.CancellationAllowed = cfg.allowed
				input.CancellationDeadlineHours = cfg.deadline
				input.RefundPolicy = cfg.policy
				input.RefundPercentage = cfg.percentage

				err := validateCreateEventInput(input)
				assert.ErrorIs(t, err, ErrInvalidCancellation)
			})
		}
	})
}

func buildValidUpdateEventInput() UpdateEventInput {
	futureDate := time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour)
	startTime := futureDate.Add(10 * time.Hour)
	endTime := futureDate.Add(12 * time.Hour)

	return UpdateEventInput{
		UserID:              uuid.New(),
		EventID:             uuid.New(),
		CategoryID:          uuid.New(),
		Title:               "Updated Party",
		Description:         "Updated description",
		Language:            "English",
		AgeRestriction:      18,
		Visibility:          "PUBLIC",
		EventType:           domain.EventTypeOnline,
		OnlineURL:           "https://meet.example.com/updated",
		EventDate:           futureDate,
		StartTime:           startTime,
		EndTime:             endTime,
		SeatLayoutType:      SeatLayoutTypeGeneral,
		BookingLimitPerUser: 5,
		TicketTypes: []TicketTypeInput{
			{
				Name:        "General Pass",
				Price:       decimal.RequireFromString("150.00"),
				Capacity:    100,
				Description: "Standard ticket",
			},
		},
		CancellationAllowed:       true,
		CancellationDeadlineHours: 24,
		RefundPolicy:              domain.RefundPolicyFull,
		RefundPercentage:          100,
	}
}

func TestValidateUpdateEventInput_CancellationSettings(t *testing.T) {
	t.Run("valid combinations", func(t *testing.T) {
		validConfigs := []struct {
			name       string
			allowed    bool
			deadline   int
			policy     string
			percentage int
		}{
			{"FULL policy with 100%", true, 24, "FULL", 100},
			{"PARTIAL policy with 75%", true, 48, "PARTIAL", 75},
			{"NO_REFUND policy with 0%", true, 24, "NO_REFUND", 0},
			{"Disabled cancellation with empty policy", false, 0, "", 0},
			{"Disabled cancellation with NO_REFUND", false, 0, "NO_REFUND", 0},
		}

		for _, cfg := range validConfigs {
			t.Run(cfg.name, func(t *testing.T) {
				input := buildValidUpdateEventInput()
				input.CancellationAllowed = cfg.allowed
				input.CancellationDeadlineHours = cfg.deadline
				input.RefundPolicy = cfg.policy
				input.RefundPercentage = cfg.percentage

				err := validateUpdateEventInput(input)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("invalid combinations return ErrInvalidCancellation", func(t *testing.T) {
		invalidConfigs := []struct {
			name       string
			allowed    bool
			deadline   int
			policy     string
			percentage int
		}{
			{"FULL with 50%", true, 24, "FULL", 50},
			{"PARTIAL with 100%", true, 24, "PARTIAL", 100},
			{"PARTIAL with 0%", true, 24, "PARTIAL", 0},
			{"NO_REFUND with 10%", true, 24, "NO_REFUND", 10},
			{"Allowed with negative deadline", true, -1, "FULL", 100},
			{"Allowed with 0 deadline", true, 0, "FULL", 100},
			{"Allowed with empty policy", true, 24, "", 100},
			{"Allowed with invalid policy", true, 24, "NONE", 0},
			{"Disabled with deadline > 0", false, 24, "", 0},
			{"Disabled with percentage > 0", false, 0, "", 20},
			{"Disabled with contradictory FULL", false, 0, "FULL", 0},
			{"Disabled with contradictory PARTIAL", false, 0, "PARTIAL", 0},
		}

		for _, cfg := range invalidConfigs {
			t.Run(cfg.name, func(t *testing.T) {
				input := buildValidUpdateEventInput()
				input.CancellationAllowed = cfg.allowed
				input.CancellationDeadlineHours = cfg.deadline
				input.RefundPolicy = cfg.policy
				input.RefundPercentage = cfg.percentage

				err := validateUpdateEventInput(input)
				assert.ErrorIs(t, err, ErrInvalidCancellation)
			})
		}
	})
}

// Mock repositories for testing UpdateEvent and PublishEvent workflows
type mockEventCancellationRepo struct {
	cancellation *domain.EventCancellation
	findErr      error
	createErr    error
	updateErr    error
	updated      *domain.EventCancellation
}

func (m *mockEventCancellationRepo) Create(c *domain.EventCancellation) error {
	m.cancellation = c
	return m.createErr
}

func (m *mockEventCancellationRepo) FindByEventID(id uuid.UUID) (*domain.EventCancellation, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.cancellation, nil
}

func (m *mockEventCancellationRepo) Update(c *domain.EventCancellation) error {
	m.updated = c
	return m.updateErr
}

func (m *mockEventCancellationRepo) DeleteByEventID(id uuid.UUID) error {
	return nil
}

type mockOrganizerRepo struct {
	domain.OrganizerRepository
	organizer *domain.Organizer
	findErr   error
}

func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.organizer, nil
}

type mockEventRepo struct {
	domain.EventRepository
	event   *domain.Event
	findErr error
	updated *domain.Event
}

func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.event, nil
}

func (m *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.event, nil
}

func (m *mockEventRepo) Update(event *domain.Event) error {
	m.updated = event
	return nil
}

type mockCategoryRepo struct {
	domain.CategoryRepository
	categories []*domain.Category
	category   *domain.Category
	findErr    error
}

func (m *mockCategoryRepo) FindActive() ([]*domain.Category, error) {
	return m.categories, m.findErr
}

func (m *mockCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) {
	return m.category, m.findErr
}

type mockScheduleRepo struct {
	domain.EventScheduleRepository
	schedule *domain.EventSchedule
	findErr  error
	updated  *domain.EventSchedule
}

func (m *mockScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	return m.schedule, m.findErr
}

func (m *mockScheduleRepo) Update(schedule *domain.EventSchedule) error {
	m.updated = schedule
	return nil
}

type mockSettingRepo struct {
	domain.EventSettingRepository
	setting *domain.EventSetting
	findErr error
	updated *domain.EventSetting
}

func (m *mockSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	return m.setting, m.findErr
}

func (m *mockSettingRepo) Update(setting *domain.EventSetting) error {
	m.updated = setting
	return nil
}

type mockTicketTypeRepo struct {
	domain.TicketTypeRepository
	ticketTypes []*domain.TicketType
	findErr     error
}

func (m *mockTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	return m.ticketTypes, m.findErr
}

func (m *mockTicketTypeRepo) Create(ctx context.Context, t *domain.TicketType) error {
	m.ticketTypes = append(m.ticketTypes, t)
	return nil
}

func (m *mockTicketTypeRepo) Update(ctx context.Context, t *domain.TicketType) error {
	return nil
}

func (m *mockTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockTestTxRepos struct {
	domain.TransactionRepositories
	organizerRepo    *mockOrganizerRepo
	eventRepo        *mockEventRepo
	categoryRepo     *mockCategoryRepo
	scheduleRepo     *mockScheduleRepo
	settingRepo      *mockSettingRepo
	cancellationRepo *mockEventCancellationRepo
	ticketTypeRepo   *mockTicketTypeRepo
}

func (r *mockTestTxRepos) OrganizerRepository() domain.OrganizerRepository {
	return r.organizerRepo
}
func (r *mockTestTxRepos) EventRepository() domain.EventRepository {
	return r.eventRepo
}
func (r *mockTestTxRepos) CategoryRepository() domain.CategoryRepository {
	return r.categoryRepo
}
func (r *mockTestTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return r.scheduleRepo
}
func (r *mockTestTxRepos) EventSettingRepository() domain.EventSettingRepository {
	return r.settingRepo
}
func (r *mockTestTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return r.cancellationRepo
}
func (r *mockTestTxRepos) TicketTypeRepository() domain.TicketTypeRepository {
	return r.ticketTypeRepo
}
func (r *mockTestTxRepos) EventContactRepository() domain.EventContactRepository {
	return &mockEventContactRepo{}
}

type mockEventContactRepo struct {
	domain.EventContactRepository
}

func (m *mockEventContactRepo) FindByEventID(eventID uuid.UUID) (*domain.EventContact, error) {
	return nil, domain.ErrEventNotFound
}

func (m *mockEventContactRepo) Create(contact *domain.EventContact) error {
	return nil
}

func (m *mockEventContactRepo) Update(contact *domain.EventContact) error {
	return nil
}

func (m *mockEventContactRepo) DeleteByEventID(id uuid.UUID) error {
	return nil
}

type mockTestTxManager struct {
	repos *mockTestTxRepos
}

func (m *mockTestTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func TestUpdateEvent_CancellationNormalizationAndPersistence(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()

	baseSetting := &domain.EventSetting{
		ID:                  uuid.New(),
		EventID:             eventID,
		SeatLayoutType:      SeatLayoutTypeGeneral,
		BookingLimitPerUser: 5,
	}

	baseSchedule := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: time.Now().Add(48 * time.Hour),
		StartTime: time.Now().Add(49 * time.Hour),
		EndTime:   time.Now().Add(51 * time.Hour),
	}

	existingCancellation := &domain.EventCancellation{
		ID:                        uuid.New(),
		EventID:                   eventID,
		CancellationAllowed:       false,
		CancellationDeadlineHours: 0,
		RefundPolicy:              domain.RefundPolicyNoRefund,
		RefundPercentage:          0,
	}

	setupUsecase := func() (*EventUsecase, *mockTestTxRepos) {
		repos := &mockTestTxRepos{
			organizerRepo: &mockOrganizerRepo{
				organizer: &domain.Organizer{
					ID:     organizerID,
					UserID: userID,
					Status: "ACTIVE",
				},
			},
			eventRepo: &mockEventRepo{
				event: &domain.Event{
					ID:          eventID,
					OrganizerID: organizerID,
					CategoryID:  categoryID,
					EventType:   domain.EventTypeOnline,
					OnlineURL:   "https://meet.example.com",
					Title:       "Sample Title",
					Description: "Sample Description",
					Status:      domain.EventStatusDraft,
				},
			},
			categoryRepo: &mockCategoryRepo{
				categories: []*domain.Category{{ID: categoryID}},
			},
			scheduleRepo: &mockScheduleRepo{
				schedule: baseSchedule,
			},
			settingRepo: &mockSettingRepo{
				setting: baseSetting,
			},
			cancellationRepo: &mockEventCancellationRepo{
				cancellation: existingCancellation,
			},
			ticketTypeRepo: &mockTicketTypeRepo{
				ticketTypes: []*domain.TicketType{
					{
						ID:                uuid.New(),
						EventID:           eventID,
						Name:              "General Pass",
						Price:             decimal.RequireFromString("150.00"),
						TotalQuantity:     100,
						AvailableQuantity: 100,
					},
				},
			},
		}

		txManager := &mockTestTxManager{repos: repos}
		uc := NewEventUsecase(txManager, nil, newTestLogger())
		return uc, repos
	}

	t.Run("successfully updates and normalizes PARTIAL cancellation policy", func(t *testing.T) {
		uc, repos := setupUsecase()

		input := buildValidUpdateEventInput()
		input.UserID = userID
		input.EventID = eventID
		input.CategoryID = categoryID
		input.CancellationAllowed = true
		input.CancellationDeadlineHours = 36
		input.RefundPolicy = "  partial  " // test whitespace and lowercase
		input.RefundPercentage = 60

		output, err := uc.UpdateEvent(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Check persisted cancellation
		persisted := repos.cancellationRepo.updated
		require.NotNil(t, persisted)
		assert.True(t, persisted.CancellationAllowed)
		assert.Equal(t, 36, persisted.CancellationDeadlineHours)
		assert.Equal(t, domain.RefundPolicyPartial, persisted.RefundPolicy)
		assert.Equal(t, 60, persisted.RefundPercentage)
	})

	t.Run("successfully updates and normalizes disabled cancellation to NO_REFUND", func(t *testing.T) {
		uc, repos := setupUsecase()

		input := buildValidUpdateEventInput()
		input.UserID = userID
		input.EventID = eventID
		input.CategoryID = categoryID
		input.CancellationAllowed = false
		input.CancellationDeadlineHours = 0
		input.RefundPolicy = ""
		input.RefundPercentage = 0

		output, err := uc.UpdateEvent(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, output)

		persisted := repos.cancellationRepo.updated
		require.NotNil(t, persisted)
		assert.False(t, persisted.CancellationAllowed)
		assert.Equal(t, 0, persisted.CancellationDeadlineHours)
		assert.Equal(t, domain.RefundPolicyNoRefund, persisted.RefundPolicy)
		assert.Equal(t, 0, persisted.RefundPercentage)
	})

	t.Run("rejects invalid cancellation without updating", func(t *testing.T) {
		uc, repos := setupUsecase()

		input := buildValidUpdateEventInput()
		input.UserID = userID
		input.EventID = eventID
		input.CategoryID = categoryID
		input.CancellationAllowed = true
		input.CancellationDeadlineHours = 24
		input.RefundPolicy = domain.RefundPolicyFull
		input.RefundPercentage = 80 // invalid for FULL (must be 100)

		output, err := uc.UpdateEvent(context.Background(), input)
		assert.ErrorIs(t, err, ErrInvalidCancellation)
		assert.Nil(t, output)
		assert.Nil(t, repos.cancellationRepo.updated)
	})
}

func TestPublishEvent_CancellationValidation(t *testing.T) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	categoryID := uuid.New()

	baseCategory := &domain.Category{
		ID:     categoryID,
		Status: domain.CategoryStatusActive,
	}

	baseSchedule := &domain.EventSchedule{
		ID:        uuid.New(),
		EventID:   eventID,
		EventDate: time.Now().Add(48 * time.Hour),
		StartTime: time.Now().Add(49 * time.Hour),
		EndTime:   time.Now().Add(51 * time.Hour),
	}

	baseSetting := &domain.EventSetting{
		ID:                  uuid.New(),
		EventID:             eventID,
		SeatLayoutType:      SeatLayoutTypeGeneral,
		BookingLimitPerUser: 5,
	}

	baseTickets := []*domain.TicketType{
		{
			ID:                uuid.New(),
			EventID:           eventID,
			Name:              "GA",
			Price:             decimal.RequireFromString("100.00"),
			TotalQuantity:     50,
			AvailableQuantity: 50,
		},
	}

	setupPublishUsecase := func(cancellation *domain.EventCancellation) (*EventUsecase, *mockTestTxRepos) {
		event := &domain.Event{
			ID:          eventID,
			OrganizerID: organizerID,
			CategoryID:  categoryID,
			EventType:   domain.EventTypeOnline,
			OnlineURL:   "https://meet.example.com",
			Title:       "Publishable Event",
			Description: "Detailed Description",
			BannerURL:   "event-banners/banner.png",
			Status:      domain.EventStatusDraft,
		}

		repos := &mockTestTxRepos{
			organizerRepo: &mockOrganizerRepo{
				organizer: &domain.Organizer{
					ID:     organizerID,
					UserID: userID,
					Status: "ACTIVE",
				},
			},
			eventRepo: &mockEventRepo{
				event: event,
			},
			categoryRepo: &mockCategoryRepo{
				category: baseCategory,
			},
			scheduleRepo: &mockScheduleRepo{
				schedule: baseSchedule,
			},
			settingRepo: &mockSettingRepo{
				setting: baseSetting,
			},
			cancellationRepo: &mockEventCancellationRepo{
				cancellation: cancellation,
			},
			ticketTypeRepo: &mockTicketTypeRepo{
				ticketTypes: baseTickets,
			},
		}

		txManager := &mockTestTxManager{repos: repos}
		uc := NewEventUsecase(txManager, nil, newTestLogger())
		return uc, repos
	}

	t.Run("publishes successfully with valid cancellation settings", func(t *testing.T) {
		validSettings := []*domain.EventCancellation{
			{
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 24,
				RefundPolicy:              domain.RefundPolicyFull,
				RefundPercentage:          100,
			},
			{
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 12,
				RefundPolicy:              domain.RefundPolicyPartial,
				RefundPercentage:          50,
			},
			{
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       false,
				CancellationDeadlineHours: 0,
				RefundPolicy:              domain.RefundPolicyNoRefund,
				RefundPercentage:          0,
			},
		}

		for _, cancellation := range validSettings {
			uc, repos := setupPublishUsecase(cancellation)
			err := uc.PublishEvent(context.Background(), userID, eventID)
			require.NoError(t, err)
			require.NotNil(t, repos.eventRepo.updated)
			assert.Equal(t, domain.EventStatusPublished, repos.eventRepo.updated.Status)
		}
	})

	t.Run("fails to publish with invalid cancellation settings", func(t *testing.T) {
		invalidSettings := []*domain.EventCancellation{
			{
				// FULL with wrong percentage
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 24,
				RefundPolicy:              domain.RefundPolicyFull,
				RefundPercentage:          50,
			},
			{
				// PARTIAL with 0 percentage
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 24,
				RefundPolicy:              domain.RefundPolicyPartial,
				RefundPercentage:          0,
			},
			{
				// NO_REFUND with percentage > 0
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       true,
				CancellationDeadlineHours: 24,
				RefundPolicy:              domain.RefundPolicyNoRefund,
				RefundPercentage:          50,
			},
			{
				// Cancellation disabled but deadline > 0
				ID:                        uuid.New(),
				EventID:                   eventID,
				CancellationAllowed:       false,
				CancellationDeadlineHours: 24,
				RefundPolicy:              domain.RefundPolicyNoRefund,
				RefundPercentage:          0,
			},
		}

		for _, cancellation := range invalidSettings {
			uc, repos := setupPublishUsecase(cancellation)
			err := uc.PublishEvent(context.Background(), userID, eventID)
			assert.ErrorIs(t, err, ErrEventNotPublishable)
			assert.Nil(t, repos.eventRepo.updated)
		}
	})
}

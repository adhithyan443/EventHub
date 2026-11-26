package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In-memory mock repositories for OrganizerBookingUsecase testing

type mockOrganizerRepo struct {
	organizersByUserID map[uuid.UUID]*domain.Organizer
	err                error
}

func (m *mockOrganizerRepo) Create(organizer *domain.Organizer) error { return nil }
func (m *mockOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	for _, o := range m.organizersByUserID {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}
func (m *mockOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	o, ok := m.organizersByUserID[userID]
	if !ok {
		return nil, domain.ErrOrganizerNotFound
	}
	return o, nil
}
func (m *mockOrganizerRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Organizer, error) {
	return m.FindByID(id)
}
func (m *mockOrganizerRepo) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	return nil, nil
}

type mockEventRepo struct {
	eventsByID map[uuid.UUID]*domain.Event
	err        error
}

func (m *mockEventRepo) Create(event *domain.Event) error { return nil }
func (m *mockEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	ev, ok := m.eventsByID[id]
	if !ok {
		return nil, domain.ErrEventNotFound
	}
	return ev, nil
}
func (m *mockEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) { return m.FindByID(id) }
func (m *mockEventRepo) Update(event *domain.Event) error                      { return nil }
func (m *mockEventRepo) Delete(id uuid.UUID) error                             { return nil }
func (m *mockEventRepo) FindByOrganizerID(organizerID uuid.UUID, page, limit int, status, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) FindAllEvents(filter domain.AdminEventFilter) ([]*domain.AdminEventListItem, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) GetAdminStatusCounts() (domain.AdminEventStatusCounts, error) {
	return domain.AdminEventStatusCounts{}, nil
}

type mockBookingRepo struct {
	listResult           *domain.OrganizerBookingListResult
	statsResult          *domain.OrganizerBookingStats
	detailResult         *domain.OrganizerBookingDetail
	adminListResult      *domain.AdminBookingListResult
	adminStatsResult     *domain.AdminBookingStats
	adminDetailResult    *domain.AdminBookingDetail
	capturedFilter       domain.OrganizerBookingFilter
	capturedAdminBFilter domain.AdminBookingFilter
	err                  error
}

func (m *mockBookingRepo) Create(ctx context.Context, booking *domain.Booking, items []domain.BookingItem) error {
	return nil
}
func (m *mockBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByBookingNumber(ctx context.Context, bookingNumber string) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByUserIDPaginated(ctx context.Context, userID uuid.UUID, filter domain.CustomerBookingFilter) ([]domain.Booking, int64, error) {
	return nil, 0, nil
}
func (m *mockBookingRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	return nil, nil, nil
}
func (m *mockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockBookingRepo) UpdateCancellationSnapshot(ctx context.Context, id uuid.UUID, allowed bool, deadlineHours int, policy string, percentage int) error {
	return nil
}
func (m *mockBookingRepo) FindItemsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.BookingItem, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindItemsByBookingIDs(ctx context.Context, bookingIDs []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) CountActiveByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockBookingRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockBookingRepo) CountConfirmedByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockBookingRepo) GetBookingStatsByEventID(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}
func (m *mockBookingRepo) GetAdminEventBookingStats(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return domain.AdminEventBookingStats{}, nil
}
func (m *mockBookingRepo) FindOrganizerBookings(ctx context.Context, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingListResult, error) {
	m.capturedFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.listResult, nil
}
func (m *mockBookingRepo) GetOrganizerBookingStats(ctx context.Context, filter domain.OrganizerBookingFilter) (*domain.OrganizerBookingStats, error) {
	m.capturedFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.statsResult, nil
}
func (m *mockBookingRepo) FindOrganizerBookingDetail(ctx context.Context, organizerID uuid.UUID, bookingID uuid.UUID) (*domain.OrganizerBookingDetail, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.detailResult == nil {
		return nil, domain.ErrBookingNotFound
	}
	return m.detailResult, nil
}
func (m *mockBookingRepo) FindAdminBookings(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingListResult, error) {
	m.capturedAdminBFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.adminListResult, nil
}
func (m *mockBookingRepo) GetAdminBookingStats(ctx context.Context, filter domain.AdminBookingFilter) (*domain.AdminBookingStats, error) {
	m.capturedAdminBFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.adminStatsResult, nil
}
func (m *mockBookingRepo) FindAdminBookingDetail(ctx context.Context, bookingID uuid.UUID) (*domain.AdminBookingDetail, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.adminDetailResult == nil {
		return nil, domain.ErrBookingNotFound
	}
	return m.adminDetailResult, nil
}

type mockTicketRepo struct {
	attendeesResult      *domain.OrganizerAttendeeListResult
	statsResult          *domain.OrganizerAttendeeStats
	adminAttendeesResult *domain.AdminAttendeeListResult
	adminStatsResult     *domain.AdminAttendeeStats
	capturedFilter       domain.OrganizerAttendeeFilter
	capturedAdminAFilter domain.AdminAttendeeFilter
	err                  error
}

func (m *mockTicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error { return nil }
func (m *mockTicketRepo) CreateMany(ctx context.Context, tickets []domain.Ticket) error {
	return nil
}
func (m *mockTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockTicketRepo) UpdateStatusForTicketIDs(ctx context.Context, ticketIDs []uuid.UUID, status string) error {
	return nil
}
func (m *mockTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	return nil
}
func (m *mockTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockTicketRepo) MarkAsUsedIfValid(ctx context.Context, ticketID, eventID uuid.UUID, usedAt time.Time) (bool, error) {
	return true, nil
}
func (m *mockTicketRepo) FindOrganizerAttendees(ctx context.Context, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeListResult, error) {
	m.capturedFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.attendeesResult, nil
}
func (m *mockTicketRepo) GetOrganizerAttendeeStats(ctx context.Context, filter domain.OrganizerAttendeeFilter) (*domain.OrganizerAttendeeStats, error) {
	m.capturedFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.statsResult, nil
}
func (m *mockTicketRepo) FindAdminAttendees(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeListResult, error) {
	m.capturedAdminAFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.adminAttendeesResult, nil
}
func (m *mockTicketRepo) GetAdminAttendeeStats(ctx context.Context, filter domain.AdminAttendeeFilter) (*domain.AdminAttendeeStats, error) {
	m.capturedAdminAFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.adminStatsResult, nil
}

func setupOrganizerTestEnv() (
	*OrganizerBookingUsecase,
	*mockBookingRepo,
	*mockTicketRepo,
	*mockOrganizerRepo,
	*mockEventRepo,
	uuid.UUID,
	*domain.Organizer,
) {
	userID := uuid.New()
	organizer := &domain.Organizer{
		ID:     uuid.New(),
		UserID: userID,
		Status: "APPROVED",
	}

	orgRepo := &mockOrganizerRepo{
		organizersByUserID: map[uuid.UUID]*domain.Organizer{
			userID: organizer,
		},
	}
	eventRepo := &mockEventRepo{
		eventsByID: make(map[uuid.UUID]*domain.Event),
	}
	bookingRepo := &mockBookingRepo{}
	ticketRepo := &mockTicketRepo{}

	uc, err := NewOrganizerBookingUsecase(bookingRepo, ticketRepo, orgRepo, eventRepo, nil)
	if err != nil {
		panic(err)
	}

	return uc, bookingRepo, ticketRepo, orgRepo, eventRepo, userID, organizer
}

// ---------------------------------------------------------------------------
// Constructor Validation Tests
// ---------------------------------------------------------------------------

func TestNewOrganizerBookingUsecase(t *testing.T) {
	bookingRepo := &mockBookingRepo{}
	ticketRepo := &mockTicketRepo{}
	orgRepo := &mockOrganizerRepo{}
	eventRepo := &mockEventRepo{}

	t.Run("nil booking repo returns error", func(t *testing.T) {
		uc, err := NewOrganizerBookingUsecase(nil, ticketRepo, orgRepo, eventRepo, nil)
		assert.Nil(t, uc)
		assert.Equal(t, ErrNilBookingRepository, err)
	})

	t.Run("nil ticket repo returns error", func(t *testing.T) {
		uc, err := NewOrganizerBookingUsecase(bookingRepo, nil, orgRepo, eventRepo, nil)
		assert.Nil(t, uc)
		assert.Equal(t, ErrNilTicketRepository, err)
	})

	t.Run("nil organizer repo returns error", func(t *testing.T) {
		uc, err := NewOrganizerBookingUsecase(bookingRepo, ticketRepo, nil, eventRepo, nil)
		assert.Nil(t, uc)
		assert.Equal(t, ErrNilOrganizerRepository, err)
	})

	t.Run("valid parameters succeed", func(t *testing.T) {
		uc, err := NewOrganizerBookingUsecase(bookingRepo, ticketRepo, orgRepo, eventRepo, nil)
		assert.NoError(t, err)
		assert.NotNil(t, uc)
	})
}

// ---------------------------------------------------------------------------
// Authorization and Organizer Status Tests
// ---------------------------------------------------------------------------

func TestOrganizerAuthorization(t *testing.T) {
	uc, _, _, orgRepo, _, userID, organizer := setupOrganizerTestEnv()
	ctx := context.Background()

	t.Run("invalid user ID", func(t *testing.T) {
		_, err := uc.ListBookings(ctx, uuid.Nil, domain.OrganizerBookingFilter{})
		assert.Equal(t, ErrInvalidUserID, err)
	})

	t.Run("organizer not found", func(t *testing.T) {
		unknownUser := uuid.New()
		_, err := uc.ListBookings(ctx, unknownUser, domain.OrganizerBookingFilter{})
		assert.Equal(t, ErrUnauthorizedOrganizer, err)
	})

	t.Run("organizer repo unexpected error", func(t *testing.T) {
		orgRepo.err = errors.New("db connection failure")
		defer func() { orgRepo.err = nil }()
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{})
		assert.EqualError(t, err, "db connection failure")
	})

	t.Run("organizer not approved or inactive", func(t *testing.T) {
		unauthorizedStatuses := []string{"PENDING", "REJECTED", "SUSPENDED", "INACTIVE", "SUBMITTED", "UNDER_REVIEW", ""}
		for _, s := range unauthorizedStatuses {
			organizer.Status = s
			_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{})
			assert.Equal(t, ErrUnauthorizedOrganizer, err, "status %s should be unauthorized", s)

			_, err = uc.GetBookingStats(ctx, userID, domain.OrganizerBookingFilter{})
			assert.Equal(t, ErrUnauthorizedOrganizer, err, "stats status %s should be unauthorized", s)

			_, err = uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{})
			assert.Equal(t, ErrUnauthorizedOrganizer, err, "attendees status %s should be unauthorized", s)

			_, err = uc.GetAttendeeStats(ctx, userID, domain.OrganizerAttendeeFilter{})
			assert.Equal(t, ErrUnauthorizedOrganizer, err, "attendee stats status %s should be unauthorized", s)

			_, err = uc.GetBookingDetail(ctx, userID, uuid.New())
			assert.Equal(t, ErrUnauthorizedOrganizer, err, "booking detail status %s should be unauthorized", s)
		}
	})

	t.Run("approved and active organizer status authorized across all operations", func(t *testing.T) {
		validStatuses := []string{"APPROVED", "ACTIVE", "approved", "active", " Approved ", " Active "}
		reqCtx := WithRequestID(ctx, "req-test-auth-123")

		for _, s := range validStatuses {
			organizer.Status = s
			_, err := uc.ListBookings(reqCtx, userID, domain.OrganizerBookingFilter{})
			assert.NoError(t, err, "ListBookings should succeed for status %s", s)

			_, err = uc.GetBookingStats(reqCtx, userID, domain.OrganizerBookingFilter{})
			assert.NoError(t, err, "GetBookingStats should succeed for status %s", s)

			_, err = uc.ListAttendees(reqCtx, userID, domain.OrganizerAttendeeFilter{})
			assert.NoError(t, err, "ListAttendees should succeed for status %s", s)

			_, err = uc.GetAttendeeStats(reqCtx, userID, domain.OrganizerAttendeeFilter{})
			assert.NoError(t, err, "GetAttendeeStats should succeed for status %s", s)
		}
	})
}

// ---------------------------------------------------------------------------
// Event Ownership Tests
// ---------------------------------------------------------------------------

func TestEventOwnership(t *testing.T) {
	uc, _, _, _, eventRepo, userID, organizer := setupOrganizerTestEnv()
	ctx := context.Background()

	ownEventID := uuid.New()
	eventRepo.eventsByID[ownEventID] = &domain.Event{
		ID:          ownEventID,
		OrganizerID: organizer.ID,
		Title:       "My Festival",
	}

	foreignEventID := uuid.New()
	eventRepo.eventsByID[foreignEventID] = &domain.Event{
		ID:          foreignEventID,
		OrganizerID: uuid.New(), // Different organizer
		Title:       "Other Event",
	}

	t.Run("nil event ID is valid (all organizer events)", func(t *testing.T) {
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{EventID: nil})
		assert.NoError(t, err)
	})

	t.Run("empty event ID returns ErrInvalidEventID", func(t *testing.T) {
		emptyID := uuid.Nil
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{EventID: &emptyID})
		assert.Equal(t, ErrInvalidEventID, err)
	})

	t.Run("event not found returns ErrEventNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{EventID: &nonExistentID})
		assert.Equal(t, ErrEventNotFound, err)
	})

	t.Run("foreign event returns ErrUnauthorizedEvent", func(t *testing.T) {
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{EventID: &foreignEventID})
		assert.Equal(t, ErrUnauthorizedEvent, err)
	})

	t.Run("owned event succeeds", func(t *testing.T) {
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{EventID: &ownEventID})
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// Filter and Pagination Normalization Tests
// ---------------------------------------------------------------------------

func TestBookingFilterNormalization(t *testing.T) {
	uc, bRepo, _, _, _, userID, organizer := setupOrganizerTestEnv()
	ctx := context.Background()

	t.Run("pagination defaults applied for zero values", func(t *testing.T) {
		// page 0 and limit 0 should get default 1 and 10
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{
			Page:  0,
			Limit: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, bRepo.capturedFilter.Page)
		assert.Equal(t, 10, bRepo.capturedFilter.Limit)
		assert.Equal(t, organizer.ID, bRepo.capturedFilter.OrganizerID)
	})

	t.Run("invalid pagination parameters rejected", func(t *testing.T) {
		invalidCases := []domain.OrganizerBookingFilter{
			{Page: -1, Limit: 10},
			{Page: 1, Limit: -5},
			{Page: 100_001, Limit: 10},
			{Page: 1, Limit: 101},
		}

		for _, tc := range invalidCases {
			_, err := uc.ListBookings(ctx, userID, tc)
			assert.Equal(t, ErrInvalidPagination, err)
		}
	})

	t.Run("booking status validation", func(t *testing.T) {
		// Valid statuses
		validStatuses := []string{"PENDING", "CONFIRMED", "CANCELLED", "PARTIALLY_CANCELLED", "all", ""}
		for _, s := range validStatuses {
			_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{BookingStatus: s})
			assert.NoError(t, err, "booking status %s should be valid", s)
		}

		// Invalid status
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{BookingStatus: "INVALID_STATUS"})
		assert.Equal(t, ErrInvalidBookingStatus, err)
	})

	t.Run("payment status validation", func(t *testing.T) {
		validStatuses := []string{"PENDING", "SUCCESS", "FAILED", "REFUNDED", "PARTIAL", "FULL", "all", ""}
		for _, s := range validStatuses {
			_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{PaymentStatus: s})
			assert.NoError(t, err, "payment status %s should be valid", s)
		}

		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{PaymentStatus: "INVALID_PAYMENT"})
		assert.Equal(t, ErrInvalidPaymentStatus, err)
	})

	t.Run("date range validation", func(t *testing.T) {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		// from before to: valid
		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{
			From: &yesterday,
			To:   &tomorrow,
		})
		assert.NoError(t, err)

		// from after to: invalid
		_, err = uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{
			From: &tomorrow,
			To:   &yesterday,
		})
		assert.Equal(t, ErrInvalidDateRange, err)
	})
}

// ---------------------------------------------------------------------------
// Booking Listing & Statistics Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookings_Success(t *testing.T) {
	uc, bRepo, _, _, _, userID, organizer := setupOrganizerTestEnv()
	ctx := context.Background()

	bRepo.listResult = &domain.OrganizerBookingListResult{
		Bookings: []domain.OrganizerBookingListItem{
			{
				ID:                   uuid.New(),
				BookingNumber:        "ORD-2026-001",
				CreatedAt:            time.Now(),
				TicketCount:          2,
				OriginalBookingTotal: decimal.NewFromInt(1500),
				Currency:             "INR",
				BookingStatus:        "CONFIRMED",
				PaymentStatus:        "SUCCESS",
			},
		},
		TotalRecords: 1,
		Page:         1,
		Limit:        10,
		TotalPages:   1,
	}

	bRepo.statsResult = &domain.OrganizerBookingStats{
		TotalBookings:       1,
		ConfirmedBookings:   1,
		TotalTicketsSold:    2,
		GrossCapturedAmount: decimal.NewFromInt(1500),
		TotalRefundedAmount: decimal.Zero,
		NetCapturedAmount:   decimal.NewFromInt(1500),
		Currency:            "INR",
	}

	t.Run("ListBookings successful", func(t *testing.T) {
		res, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Len(t, res.Bookings, 1)
		assert.Equal(t, "ORD-2026-001", res.Bookings[0].BookingNumber)
		assert.Equal(t, organizer.ID, bRepo.capturedFilter.OrganizerID)
	})

	t.Run("GetBookingStats successful", func(t *testing.T) {
		stats, err := uc.GetBookingStats(ctx, userID, domain.OrganizerBookingFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.TotalBookings)
		assert.Equal(t, int64(2), stats.TotalTicketsSold)
		assert.True(t, stats.NetCapturedAmount.Equal(decimal.NewFromInt(1500)))
	})

	t.Run("Repository error propagated", func(t *testing.T) {
		bRepo.err = errors.New("repository query error")
		defer func() { bRepo.err = nil }()

		_, err := uc.ListBookings(ctx, userID, domain.OrganizerBookingFilter{})
		assert.EqualError(t, err, "repository query error")

		_, err = uc.GetBookingStats(ctx, userID, domain.OrganizerBookingFilter{})
		assert.EqualError(t, err, "repository query error")
	})
}

// ---------------------------------------------------------------------------
// Booking Detail Tests
// ---------------------------------------------------------------------------

func TestOrganizerBookingDetail(t *testing.T) {
	uc, bRepo, _, _, _, userID, _ := setupOrganizerTestEnv()
	ctx := context.Background()

	bookingID := uuid.New()
	bRepo.detailResult = &domain.OrganizerBookingDetail{
		ID:            bookingID,
		BookingNumber: "ORD-2026-099",
		BookingStatus: "CONFIRMED",
		Financials: domain.OrganizerBookingFinancials{
			OriginalBookingTotal:     decimal.NewFromInt(2000),
			SuccessfulCapturedAmount: decimal.NewFromInt(2000),
			NetCapturedAfterRefunds:  decimal.NewFromInt(2000),
			Currency:                 "INR",
		},
	}

	t.Run("invalid booking ID", func(t *testing.T) {
		_, err := uc.GetBookingDetail(ctx, userID, uuid.Nil)
		assert.Equal(t, ErrInvalidBookingID, err)
	})

	t.Run("booking not found", func(t *testing.T) {
		bRepo.detailResult = nil
		_, err := uc.GetBookingDetail(ctx, userID, uuid.New())
		assert.Equal(t, domain.ErrBookingNotFound, err)
	})

	t.Run("successful retrieval", func(t *testing.T) {
		bRepo.detailResult = &domain.OrganizerBookingDetail{
			ID:            bookingID,
			BookingNumber: "ORD-2026-099",
			BookingStatus: "CONFIRMED",
		}
		res, err := uc.GetBookingDetail(ctx, userID, bookingID)
		require.NoError(t, err)
		assert.Equal(t, bookingID, res.ID)
		assert.Equal(t, "ORD-2026-099", res.BookingNumber)
	})
}

// ---------------------------------------------------------------------------
// Attendee Listing & Statistics Tests
// ---------------------------------------------------------------------------

func TestOrganizerAttendees(t *testing.T) {
	uc, _, tRepo, _, _, userID, organizer := setupOrganizerTestEnv()
	ctx := context.Background()

	tRepo.attendeesResult = &domain.OrganizerAttendeeListResult{
		Attendees: []domain.OrganizerAttendeeListItem{
			{
				TicketID:        uuid.New(),
				TicketCode:      "TCK-2026-A1",
				AdmissionNumber: 1,
				TicketTypeName:  "VIP Access",
				Status:          "VALID",
				Price:           decimal.NewFromInt(1000),
			},
		},
		TotalRecords: 1,
		Page:         1,
		Limit:        10,
		TotalPages:   1,
	}

	tRepo.statsResult = &domain.OrganizerAttendeeStats{
		TotalIssuedTickets:       10,
		ActiveEligibleAdmissions: 9,
		CheckedInTickets:         3,
		PendingCheckInTickets:    6,
		CancelledTickets:         1,
		CheckInRate:              decimal.NewFromFloat(33.33),
	}

	t.Run("attendee pagination defaults and validation", func(t *testing.T) {
		// Defaulting zero values
		_, err := uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{
			Page:  0,
			Limit: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, tRepo.capturedFilter.Page)
		assert.Equal(t, 10, tRepo.capturedFilter.Limit)

		// Invalid pagination rejected
		invalidCases := []domain.OrganizerAttendeeFilter{
			{Page: -1, Limit: 10},
			{Page: 1, Limit: -1},
			{Page: 100_001, Limit: 10},
			{Page: 1, Limit: 150},
		}
		for _, tc := range invalidCases {
			_, err := uc.ListAttendees(ctx, userID, tc)
			assert.Equal(t, ErrInvalidPagination, err)
		}
	})

	t.Run("check-in status filter validation", func(t *testing.T) {
		validStatuses := []string{"USED", "CHECKED_IN", "VALID", "PENDING", "CANCELLED", "REFUNDED", "CANCELLED_REFUNDED", "all", ""}
		for _, s := range validStatuses {
			_, err := uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{CheckInStatus: s})
			assert.NoError(t, err, "check-in status %s should be valid", s)
		}

		_, err := uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{CheckInStatus: "UNKNOWN_STATUS"})
		assert.Equal(t, ErrInvalidCheckInStatus, err)
	})

	t.Run("successful attendee listing", func(t *testing.T) {
		res, err := uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalRecords)
		assert.Len(t, res.Attendees, 1)
		assert.Equal(t, "TCK-2026-A1", res.Attendees[0].TicketCode)
		assert.Equal(t, organizer.ID, tRepo.capturedFilter.OrganizerID)
	})

	t.Run("successful attendee stats", func(t *testing.T) {
		stats, err := uc.GetAttendeeStats(ctx, userID, domain.OrganizerAttendeeFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(10), stats.TotalIssuedTickets)
		assert.Equal(t, int64(3), stats.CheckedInTickets)
		assert.True(t, stats.CheckInRate.Equal(decimal.NewFromFloat(33.33)))
	})

	t.Run("repository error on attendee listing", func(t *testing.T) {
		tRepo.err = errors.New("attendee repo failed")
		defer func() { tRepo.err = nil }()

		_, err := uc.ListAttendees(ctx, userID, domain.OrganizerAttendeeFilter{})
		assert.EqualError(t, err, "attendee repo failed")

		_, err = uc.GetAttendeeStats(ctx, userID, domain.OrganizerAttendeeFilter{})
		assert.EqualError(t, err, "attendee repo failed")
	})
}

func TestAdminBookingUsecase(t *testing.T) {
	bRepo := &mockBookingRepo{}
	tRepo := &mockTicketRepo{}
	u, err := NewAdminBookingUsecase(bRepo, tRepo, nil, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("list bookings success", func(t *testing.T) {
		bRepo.adminListResult = &domain.AdminBookingListResult{
			Bookings: []domain.AdminBookingListItem{
				{
					ID:            uuid.New(),
					BookingNumber: "BK-ADM-1",
					Organizer: domain.AdminOrganizerBrief{
						BusinessName: "Grand Shows",
					},
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}

		res, err := u.ListBookings(ctx, domain.AdminBookingFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, len(res.Bookings))
		assert.Equal(t, "BK-ADM-1", res.Bookings[0].BookingNumber)
		assert.Equal(t, "Grand Shows", res.Bookings[0].Organizer.BusinessName)
	})

	t.Run("list bookings validation", func(t *testing.T) {
		from := time.Now()
		to := from.Add(-1 * time.Hour)
		_, err := u.ListBookings(ctx, domain.AdminBookingFilter{Page: 1, Limit: 10, From: &from, To: &to})
		assert.ErrorIs(t, err, ErrInvalidDateRange)

		_, err = u.ListBookings(ctx, domain.AdminBookingFilter{Page: 1, Limit: 10, BookingStatus: "INVALID"})
		assert.ErrorIs(t, err, ErrInvalidBookingStatus)

		_, err = u.ListBookings(ctx, domain.AdminBookingFilter{Page: 1, Limit: 10, PaymentStatus: "INVALID"})
		assert.ErrorIs(t, err, ErrInvalidPaymentStatus)

		_, err = u.ListBookings(ctx, domain.AdminBookingFilter{Page: 0, Limit: 10})
		assert.ErrorIs(t, err, ErrInvalidPagination)
	})

	t.Run("booking stats", func(t *testing.T) {
		bRepo.adminStatsResult = &domain.AdminBookingStats{
			TotalBookings:     100,
			ConfirmedBookings: 90,
			CancelledBookings: 10,
		}

		res, err := u.GetBookingStats(ctx, domain.AdminBookingFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(100), res.TotalBookings)
	})

	t.Run("booking detail", func(t *testing.T) {
		_, err := u.GetBookingDetail(ctx, uuid.Nil)
		assert.ErrorIs(t, err, ErrInvalidBookingID)

		bID := uuid.New()
		bRepo.adminDetailResult = &domain.AdminBookingDetail{
			ID:            bID,
			BookingNumber: "BK-ADM-DETAIL",
		}
		detail, err := u.GetBookingDetail(ctx, bID)
		require.NoError(t, err)
		assert.Equal(t, "BK-ADM-DETAIL", detail.BookingNumber)
	})

	t.Run("attendees list and stats", func(t *testing.T) {
		tRepo.adminAttendeesResult = &domain.AdminAttendeeListResult{
			Attendees: []domain.AdminAttendeeListItem{
				{
					TicketID:   uuid.New(),
					TicketCode: "TCK-ADM-1",
				},
			},
			TotalRecords: 1,
			Page:         1,
			Limit:        10,
			TotalPages:   1,
		}

		res, err := u.ListAttendees(ctx, domain.AdminAttendeeFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, len(res.Attendees))

		_, err = u.ListAttendees(ctx, domain.AdminAttendeeFilter{Page: 1, Limit: 10, CheckInStatus: "INVALID"})
		assert.ErrorIs(t, err, ErrInvalidCheckInStatus)

		tRepo.adminStatsResult = &domain.AdminAttendeeStats{
			TotalIssuedTickets: 200,
		}
		stats, err := u.GetAttendeeStats(ctx, domain.AdminAttendeeFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(200), stats.TotalIssuedTickets)
	})
}

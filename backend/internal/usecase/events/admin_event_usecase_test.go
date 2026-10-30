package events

import (
	"context"
	"errors"
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

// --- Mock Implementations for AdminEventUsecase Tests ---

type mockTxManager struct {
	txRepos *mockAdminTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

type mockAdminTxRepos struct {
	domain.TransactionRepositories
	eventRepo            domain.EventRepository
	bookingRepo          domain.BookingRepository
	reservationRepo      domain.SeatReservationRepository
	organizerRepo        domain.OrganizerRepository
	organizerProfileRepo domain.OrganizerProfileRepository
	userRepo             domain.UserRepository
	categoryRepo         domain.CategoryRepository
	venueRepo            domain.VenueRepository
	scheduleRepo         domain.EventScheduleRepository
	settingRepo          domain.EventSettingRepository
	cancellationRepo     domain.EventCancellationRepository
	contactRepo          domain.EventContactRepository
	ticketTypeRepo       domain.TicketTypeRepository
	ticketRepo           domain.TicketRepository
	seatLayoutRepo       domain.SeatLayoutRepository
	seatSectionRepo      domain.SeatSectionRepository
}

func (m *mockAdminTxRepos) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockAdminTxRepos) BookingRepository() domain.BookingRepository { return m.bookingRepo }
func (m *mockAdminTxRepos) TicketRepository() domain.TicketRepository {
	if m.ticketRepo == nil {
		return &mockAdminTicketRepo{}
	}
	return m.ticketRepo
}
func (m *mockAdminTxRepos) SeatReservationRepository() domain.SeatReservationRepository {
	return m.reservationRepo
}
func (m *mockAdminTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockAdminTxRepos) OrganizerProfileRepository() domain.OrganizerProfileRepository {
	return m.organizerProfileRepo
}
func (m *mockAdminTxRepos) UserRepository() domain.UserRepository { return m.userRepo }
func (m *mockAdminTxRepos) CategoryRepository() domain.CategoryRepository { return m.categoryRepo }
func (m *mockAdminTxRepos) VenueRepository() domain.VenueRepository { return m.venueRepo }
func (m *mockAdminTxRepos) EventScheduleRepository() domain.EventScheduleRepository {
	return m.scheduleRepo
}
func (m *mockAdminTxRepos) EventSettingRepository() domain.EventSettingRepository {
	return m.settingRepo
}
func (m *mockAdminTxRepos) EventCancellationRepository() domain.EventCancellationRepository {
	return m.cancellationRepo
}
func (m *mockAdminTxRepos) EventContactRepository() domain.EventContactRepository {
	return m.contactRepo
}
func (m *mockAdminTxRepos) TicketTypeRepository() domain.TicketTypeRepository { return m.ticketTypeRepo }
func (m *mockAdminTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository { return m.seatLayoutRepo }
func (m *mockAdminTxRepos) SeatSectionRepository() domain.SeatSectionRepository {
	return m.seatSectionRepo
}

type mockAdminEventRepo struct {
	events       map[uuid.UUID]*domain.Event
	listItems    []*domain.AdminEventListItem
	statusCounts domain.AdminEventStatusCounts
	deletedIDs   []uuid.UUID
}

func (m *mockAdminEventRepo) Create(event *domain.Event) error { return nil }
func (m *mockAdminEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if ev, ok := m.events[id]; ok {
		copied := *ev
		return &copied, nil
	}
	return nil, domain.ErrEventNotFound
}
func (m *mockAdminEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	return m.FindByID(id)
}
func (m *mockAdminEventRepo) Update(event *domain.Event) error {
	if _, ok := m.events[event.ID]; !ok {
		return domain.ErrEventNotFound
	}
	copied := *event
	m.events[event.ID] = &copied
	return nil
}
func (m *mockAdminEventRepo) Delete(id uuid.UUID) error {
	delete(m.events, id)
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}
func (m *mockAdminEventRepo) FindByOrganizerID(orgID uuid.UUID, page, limit int, status, search string) ([]*domain.OrganizerEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockAdminEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	return nil, 0, nil
}
func (m *mockAdminEventRepo) FindAllEvents(filter domain.AdminEventFilter) ([]*domain.AdminEventListItem, int64, error) {
	return m.listItems, int64(len(m.listItems)), nil
}
func (m *mockAdminEventRepo) GetAdminStatusCounts() (domain.AdminEventStatusCounts, error) {
	return m.statusCounts, nil
}

type mockAdminBookingRepo struct {
	confirmedCount int64
	totalCount     int64
	stats          domain.AdminEventBookingStats
}

func (m *mockAdminBookingRepo) Create(ctx context.Context, b *domain.Booking, items []domain.BookingItem) error {
	return nil
}
func (m *mockAdminBookingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindByBookingNumber(ctx context.Context, num string) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindByReservationID(ctx context.Context, resID uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindWithItemsByID(ctx context.Context, id uuid.UUID) (*domain.Booking, []domain.BookingItem, error) {
	return nil, nil, nil
}
func (m *mockAdminBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminBookingRepo) ConfirmPending(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockAdminBookingRepo) FindItemsByBookingID(ctx context.Context, id uuid.UUID) ([]domain.BookingItem, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) FindItemsByBookingIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]domain.BookingItem, error) {
	return nil, nil
}
func (m *mockAdminBookingRepo) CountConfirmedByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return m.confirmedCount, nil
}
func (m *mockAdminBookingRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	if m.totalCount != 0 {
		return m.totalCount, nil
	}
	return m.confirmedCount, nil
}
func (m *mockAdminBookingRepo) GetBookingStatsByEventID(ctx context.Context, eventID uuid.UUID) (domain.AdminEventBookingStats, error) {
	return m.stats, nil
}

type mockAdminTicketRepo struct {
	ticketCount int64
}

func (m *mockAdminTicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error { return nil }
func (m *mockAdminTicketRepo) CreateMany(ctx context.Context, tickets []domain.Ticket) error {
	return nil
}
func (m *mockAdminTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockAdminTicketRepo) FindByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	return nil, nil
}
func (m *mockAdminTicketRepo) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockAdminTicketRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Ticket, error) {
	return nil, nil
}
func (m *mockAdminTicketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminTicketRepo) UpdateStatusByBookingID(ctx context.Context, bookingID uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminTicketRepo) CountByEventID(ctx context.Context, eventID uuid.UUID) (int64, error) {
	return m.ticketCount, nil
}

type mockAdminReservationRepo struct {
	activeCount int64
}

func (m *mockAdminReservationRepo) Create(ctx context.Context, r *domain.SeatReservation) error {
	return nil
}
func (m *mockAdminReservationRepo) CreateItem(ctx context.Context, item *domain.SeatReservationItem) error {
	return nil
}
func (m *mockAdminReservationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	return nil, nil
}
func (m *mockAdminReservationRepo) FindItemsByReservationID(ctx context.Context, id uuid.UUID) ([]domain.SeatReservationItem, error) {
	return nil, nil
}
func (m *mockAdminReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminReservationRepo) CompleteIfActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockAdminReservationRepo) ExpireIfActive(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return true, nil
}
func (m *mockAdminReservationRepo) FindExpiredActive(ctx context.Context, now time.Time, limit int) ([]domain.SeatReservation, error) {
	return nil, nil
}
func (m *mockAdminReservationRepo) CancelIfActive(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockAdminReservationRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SeatReservation, error) {
	return nil, nil
}
func (m *mockAdminReservationRepo) CountActiveByEventID(ctx context.Context, eventID uuid.UUID, now time.Time) (int64, error) {
	return m.activeCount, nil
}

type mockAdminOrganizerRepo struct {
	organizers map[uuid.UUID]*domain.Organizer
}

func (m *mockAdminOrganizerRepo) Create(org *domain.Organizer) error { return nil }
func (m *mockAdminOrganizerRepo) FindByID(id uuid.UUID) (*domain.Organizer, error) {
	if org, ok := m.organizers[id]; ok {
		return org, nil
	}
	return nil, domain.ErrOrganizerNotFound
}
func (m *mockAdminOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	for _, org := range m.organizers {
		if org.UserID == userID {
			return org, nil
		}
	}
	return nil, domain.ErrOrganizerNotFound
}
func (m *mockAdminOrganizerRepo) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	return nil, nil
}

type mockAdminOrganizerProfileRepo struct {
	profiles map[uuid.UUID]*domain.OrganizerProfile
}

func (m *mockAdminOrganizerProfileRepo) Create(p *domain.OrganizerProfile) error { return nil }
func (m *mockAdminOrganizerProfileRepo) FindByOrganizerID(orgID uuid.UUID) (*domain.OrganizerProfile, error) {
	if p, ok := m.profiles[orgID]; ok {
		return p, nil
	}
	return nil, errors.New("profile not found")
}
func (m *mockAdminOrganizerProfileRepo) Update(p *domain.OrganizerProfile) error { return nil }

type mockAdminCategoryRepo struct {
	categories map[uuid.UUID]*domain.Category
}

func (m *mockAdminCategoryRepo) Create(c *domain.Category) error { return nil }
func (m *mockAdminCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) {
	if c, ok := m.categories[id]; ok {
		return c, nil
	}
	return nil, domain.ErrCategoryNotFound
}
func (m *mockAdminCategoryRepo) FindAll() ([]*domain.Category, error)             { return nil, nil }
func (m *mockAdminCategoryRepo) FindActive() ([]*domain.Category, error)          { return nil, nil }
func (m *mockAdminCategoryRepo) FindByName(name string) (*domain.Category, error) { return nil, nil }
func (m *mockAdminCategoryRepo) Update(c *domain.Category) error                  { return nil }
func (m *mockAdminCategoryRepo) Delete(id uuid.UUID) error                        { return nil }

type mockAdminScheduleRepo struct {
	schedules map[uuid.UUID]*domain.EventSchedule
}

func (m *mockAdminScheduleRepo) Create(s *domain.EventSchedule) error {
	m.schedules[s.EventID] = s
	return nil
}
func (m *mockAdminScheduleRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSchedule, error) {
	if s, ok := m.schedules[eventID]; ok {
		return s, nil
	}
	return nil, nil
}
func (m *mockAdminScheduleRepo) Update(s *domain.EventSchedule) error {
	m.schedules[s.EventID] = s
	return nil
}
func (m *mockAdminScheduleRepo) DeleteByEventID(eventID uuid.UUID) error {
	delete(m.schedules, eventID)
	return nil
}

type mockAdminSettingRepo struct {
	settings map[uuid.UUID]*domain.EventSetting
}

func (m *mockAdminSettingRepo) Create(s *domain.EventSetting) error {
	m.settings[s.EventID] = s
	return nil
}
func (m *mockAdminSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	if s, ok := m.settings[eventID]; ok {
		return s, nil
	}
	return nil, nil
}
func (m *mockAdminSettingRepo) Update(s *domain.EventSetting) error {
	m.settings[s.EventID] = s
	return nil
}
func (m *mockAdminSettingRepo) DeleteByEventID(eventID uuid.UUID) error {
	delete(m.settings, eventID)
	return nil
}

type mockAdminCancellationRepo struct {
	cancellations map[uuid.UUID]*domain.EventCancellation
}

func (m *mockAdminCancellationRepo) Create(c *domain.EventCancellation) error {
	m.cancellations[c.EventID] = c
	return nil
}
func (m *mockAdminCancellationRepo) FindByEventID(eventID uuid.UUID) (*domain.EventCancellation, error) {
	if c, ok := m.cancellations[eventID]; ok {
		return c, nil
	}
	return nil, nil
}
func (m *mockAdminCancellationRepo) Update(c *domain.EventCancellation) error {
	m.cancellations[c.EventID] = c
	return nil
}
func (m *mockAdminCancellationRepo) DeleteByEventID(eventID uuid.UUID) error {
	delete(m.cancellations, eventID)
	return nil
}

type mockAdminContactRepo struct {
	contacts map[uuid.UUID]*domain.EventContact
}

func (m *mockAdminContactRepo) Create(c *domain.EventContact) error {
	m.contacts[c.EventID] = c
	return nil
}
func (m *mockAdminContactRepo) FindByEventID(eventID uuid.UUID) (*domain.EventContact, error) {
	if c, ok := m.contacts[eventID]; ok {
		return c, nil
	}
	return nil, nil
}
func (m *mockAdminContactRepo) Update(c *domain.EventContact) error {
	m.contacts[c.EventID] = c
	return nil
}
func (m *mockAdminContactRepo) DeleteByEventID(eventID uuid.UUID) error {
	delete(m.contacts, eventID)
	return nil
}

type mockAdminTicketTypeRepo struct {
	types map[uuid.UUID][]*domain.TicketType
}

func (m *mockAdminTicketTypeRepo) Create(ctx context.Context, t *domain.TicketType) error { return nil }
func (m *mockAdminTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	return nil, nil
}
func (m *mockAdminTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	return m.types[eventID], nil
}
func (m *mockAdminTicketTypeRepo) Update(ctx context.Context, t *domain.TicketType) error { return nil }
func (m *mockAdminTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *mockAdminTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	delete(m.types, eventID)
	return nil
}
func (m *mockAdminTicketTypeRepo) ReserveQuantityIfAvailable(ctx context.Context, id uuid.UUID, q int) error {
	return nil
}
func (m *mockAdminTicketTypeRepo) ReleaseReservedQuantity(ctx context.Context, id uuid.UUID, q int) error {
	return nil
}

type mockAdminSeatLayoutRepo struct {
	layouts map[uuid.UUID]*domain.SeatLayout
}

func (m *mockAdminSeatLayoutRepo) Create(l *domain.SeatLayout) error { return nil }
func (m *mockAdminSeatLayoutRepo) Update(l *domain.SeatLayout) error { return nil }
func (m *mockAdminSeatLayoutRepo) FindByEventID(eventID uuid.UUID) (*domain.SeatLayout, error) {
	if l, ok := m.layouts[eventID]; ok {
		return l, nil
	}
	return nil, domain.ErrSeatLayoutNotFound
}
func (m *mockAdminSeatLayoutRepo) DeleteByEventID(eventID uuid.UUID) error {
	delete(m.layouts, eventID)
	return nil
}

type mockAdminSeatSectionRepo struct {
	sections map[uuid.UUID][]domain.SeatSectionWithStats
}

func (m *mockAdminSeatSectionRepo) Create(s *domain.SeatSection) error { return nil }
func (m *mockAdminSeatSectionRepo) Update(s *domain.SeatSection) error { return nil }
func (m *mockAdminSeatSectionRepo) Delete(id uuid.UUID) error          { return nil }
func (m *mockAdminSeatSectionRepo) DeleteBySeatLayoutID(id uuid.UUID) error {
	delete(m.sections, id)
	return nil
}
func (m *mockAdminSeatSectionRepo) FindBySeatLayoutID(id uuid.UUID) ([]domain.SeatSection, error) {
	return nil, nil
}
func (m *mockAdminSeatSectionRepo) FindWithStatsBySeatLayoutID(id uuid.UUID) ([]domain.SeatSectionWithStats, error) {
	return m.sections[id], nil
}
func (m *mockAdminSeatSectionRepo) FindByID(id uuid.UUID) (*domain.SeatSection, error) {
	return nil, nil
}

type mockAdminUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (m *mockAdminUserRepo) Create(u *domain.User) error { return nil }
func (m *mockAdminUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockAdminUserRepo) FindByEmail(email string) (*domain.User, error) { return nil, nil }
func (m *mockAdminUserRepo) FindByPhone(phone string) (*domain.User, error) { return nil, nil }
func (m *mockAdminUserRepo) Update(u *domain.User) error                    { return nil }
func (m *mockAdminUserRepo) List(filter domain.UserListFilter) (*domain.UserList, error) {
	return nil, nil
}

type mockAdminVenueRepo struct {
	venues map[uuid.UUID]*domain.Venue
}

func (m *mockAdminVenueRepo) Create(v *domain.Venue) error { return nil }
func (m *mockAdminVenueRepo) FindByID(id uuid.UUID) (*domain.Venue, error) {
	if v, ok := m.venues[id]; ok {
		return v, nil
	}
	return nil, domain.ErrVenueNotFound
}
func (m *mockAdminVenueRepo) FindByGooglePlaceID(id string) (*domain.Venue, error) {
	return nil, domain.ErrVenueNotFound
}
func (m *mockAdminVenueRepo) FindAll(page, limit int) ([]*domain.Venue, int64, error) {
	return nil, 0, nil
}

// --- Test Cases ---

func TestAdminEventUsecase_ListEvents(t *testing.T) {
	eventRepo := &mockAdminEventRepo{
		listItems: []*domain.AdminEventListItem{
			{
				ID:                    uuid.New(),
				Title:                 "Tech Summit 2026",
				OrganizerBusinessName: "TechCorp",
				Status:                domain.EventStatusPublished,
			},
		},
		statusCounts: domain.AdminEventStatusCounts{
			Total:     10,
			Published: 6,
			Draft:     3,
			Cancelled: 1,
		},
	}

	uc := NewAdminEventUsecase(nil, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	res, err := uc.ListEvents(context.Background(), domain.AdminEventFilter{Page: 1, Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Len(t, res.Events, 1)
	assert.Equal(t, "Tech Summit 2026", res.Events[0].Title)
	assert.Equal(t, int64(10), res.StatusCounts.Total)
	assert.Equal(t, int64(6), res.StatusCounts.Published)
}

func TestAdminEventUsecase_GetEventDetails(t *testing.T) {
	eventID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()
	catID := uuid.New()

	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:          eventID,
				OrganizerID: orgID,
				CategoryID:  catID,
				Title:       "Concert Fest",
				Status:      domain.EventStatusPublished,
			},
		},
	}

	orgRepo := &mockAdminOrganizerRepo{
		organizers: map[uuid.UUID]*domain.Organizer{
			orgID: {ID: orgID, UserID: userID, Status: "ACTIVE"},
		},
	}
	orgProfileRepo := &mockAdminOrganizerProfileRepo{
		profiles: map[uuid.UUID]*domain.OrganizerProfile{
			orgID: {OrganizerID: orgID, BusinessName: "LiveMusic Inc", Email: "contact@livemusic.com"},
		},
	}
	userRepo := &mockAdminUserRepo{
		users: map[uuid.UUID]*domain.User{
			userID: {ID: userID, Email: "owner@livemusic.com", Phone: "1234567890"},
		},
	}
	catRepo := &mockAdminCategoryRepo{
		categories: map[uuid.UUID]*domain.Category{
			catID: {ID: catID, Name: "Music", Status: domain.CategoryStatusActive},
		},
	}
	bookingRepo := &mockAdminBookingRepo{
		stats: domain.AdminEventBookingStats{
			TotalBookings:     5,
			ConfirmedBookings: 4,
			CancelledBookings: 1,
			TicketsSold:       12,
			TotalRevenue:      decimal.NewFromInt(1200),
		},
	}
	resRepo := &mockAdminReservationRepo{activeCount: 2}

	txRepos := &mockAdminTxRepos{
		eventRepo:            eventRepo,
		organizerRepo:        orgRepo,
		organizerProfileRepo: orgProfileRepo,
		userRepo:             userRepo,
		categoryRepo:         catRepo,
		bookingRepo:          bookingRepo,
		reservationRepo:      resRepo,
		scheduleRepo:         &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:          &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo:     &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:          &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
		ticketTypeRepo:       &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:       &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		venueRepo:            &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	details, err := uc.GetEventDetails(context.Background(), eventID)

	require.NoError(t, err)
	require.NotNil(t, details)
	assert.Equal(t, "Concert Fest", details.Event.Title)
	assert.Equal(t, "LiveMusic Inc", details.Organizer.BusinessName)
	assert.Equal(t, "Music", details.Category.Name)
	assert.Equal(t, int64(4), details.BookingStats.ConfirmedBookings)
	assert.Equal(t, int64(2), details.ActiveReservations)
}

func TestAdminEventUsecase_UpdateEvent_DraftFullEdit(t *testing.T) {
	eventID := uuid.New()
	catID := uuid.New()

	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:         eventID,
				CategoryID: catID,
				Title:      "Original Title",
				Status:     domain.EventStatusDraft,
			},
		},
	}
	catRepo := &mockAdminCategoryRepo{
		categories: map[uuid.UUID]*domain.Category{
			catID: {ID: catID, Name: "Tech", Status: domain.CategoryStatusActive},
		},
	}

	txRepos := &mockAdminTxRepos{
		eventRepo:            eventRepo,
		categoryRepo:         catRepo,
		scheduleRepo:         &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:          &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo:     &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:          &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
		organizerRepo:        &mockAdminOrganizerRepo{organizers: map[uuid.UUID]*domain.Organizer{}},
		organizerProfileRepo: &mockAdminOrganizerProfileRepo{profiles: map[uuid.UUID]*domain.OrganizerProfile{}},
		userRepo:             &mockAdminUserRepo{users: map[uuid.UUID]*domain.User{}},
		bookingRepo:          &mockAdminBookingRepo{},
		reservationRepo:      &mockAdminReservationRepo{},
		ticketTypeRepo:       &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:       &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		venueRepo:            &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Now().Add(48 * time.Hour)
	limit := 5

	updated, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
		EventID:             eventID,
		Title:               "Updated Draft Title",
		Description:         "New Description",
		EventDate:           &now,
		BookingLimitPerUser: &limit,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Draft Title", updated.Event.Title)
	assert.Equal(t, "New Description", updated.Event.Description)
}

func TestAdminEventUsecase_UpdateEvent_PublishedDescriptiveOnly(t *testing.T) {
	eventID := uuid.New()
	catID := uuid.New()

	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:         eventID,
				CategoryID: catID,
				Title:      "Original Published Title",
				Status:     domain.EventStatusPublished,
			},
		},
	}

	txRepos := &mockAdminTxRepos{
		eventRepo:            eventRepo,
		categoryRepo:         &mockAdminCategoryRepo{categories: map[uuid.UUID]*domain.Category{catID: {ID: catID}}},
		scheduleRepo:         &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:          &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo:     &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:          &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
		organizerRepo:        &mockAdminOrganizerRepo{organizers: map[uuid.UUID]*domain.Organizer{}},
		organizerProfileRepo: &mockAdminOrganizerProfileRepo{profiles: map[uuid.UUID]*domain.OrganizerProfile{}},
		userRepo:             &mockAdminUserRepo{users: map[uuid.UUID]*domain.User{}},
		bookingRepo:          &mockAdminBookingRepo{},
		reservationRepo:      &mockAdminReservationRepo{},
		ticketTypeRepo:       &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:       &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		venueRepo:            &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// 1. Modifying descriptive metadata succeeds
	updated, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
		EventID:     eventID,
		Title:       "Updated Published Title",
		Description: "Updated Published Description",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Published Title", updated.Event.Title)

	// 2. Modifying structural field (e.g. Schedule Date) fails with ErrPublishedEventStructuralEdit
	future := time.Now().Add(72 * time.Hour)
	_, err = uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
		EventID:   eventID,
		EventDate: &future,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPublishedEventStructuralEdit)
}

func TestAdminEventUsecase_UpdateEvent_CancelledForbidden(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Cancelled Event",
				Status: domain.EventStatusCancelled,
			},
		},
	}

	txRepos := &mockAdminTxRepos{eventRepo: eventRepo}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
		EventID: eventID,
		Title:   "Attempted Rename",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCancelledEventNotEditable)
}

func TestAdminEventUsecase_DeleteEvent_ZeroBookingsSuccess(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Empty Event",
				Status: domain.EventStatusPublished,
			},
		},
	}

	bookingRepo := &mockAdminBookingRepo{confirmedCount: 0}
	resRepo := &mockAdminReservationRepo{activeCount: 0}

	txRepos := &mockAdminTxRepos{
		eventRepo:        eventRepo,
		bookingRepo:      bookingRepo,
		reservationRepo:  resRepo,
		ticketTypeRepo:   &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:   &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		scheduleRepo:     &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:      &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo: &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:      &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := uc.DeleteEvent(context.Background(), uuid.New(), eventID)
	require.NoError(t, err)
	assert.Contains(t, eventRepo.deletedIDs, eventID)
}

func TestAdminEventUsecase_DeleteEvent_HasBookingsConflict(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Booked Event",
				Status: domain.EventStatusPublished,
			},
		},
	}

	bookingRepo := &mockAdminBookingRepo{confirmedCount: 3}
	resRepo := &mockAdminReservationRepo{activeCount: 0}

	txRepos := &mockAdminTxRepos{
		eventRepo:       eventRepo,
		bookingRepo:     bookingRepo,
		reservationRepo: resRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := uc.DeleteEvent(context.Background(), uuid.New(), eventID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventCannotBeDeletedWithBookings)
	assert.NotContains(t, eventRepo.deletedIDs, eventID)
}

func TestAdminEventUsecase_DeleteEvent_HasActiveReservationsConflict(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Reserved Event",
				Status: domain.EventStatusPublished,
			},
		},
	}

	bookingRepo := &mockAdminBookingRepo{confirmedCount: 0}
	resRepo := &mockAdminReservationRepo{activeCount: 1} // 1 active reservation hold

	txRepos := &mockAdminTxRepos{
		eventRepo:       eventRepo,
		bookingRepo:     bookingRepo,
		reservationRepo: resRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := uc.DeleteEvent(context.Background(), uuid.New(), eventID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventCannotBeDeletedWithBookings)
	assert.NotContains(t, eventRepo.deletedIDs, eventID)
}

func TestAdminEventUsecase_DeleteEvent_HasHistoricalCancelledBookingsConflict(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Cancelled Bookings Event",
				Status: domain.EventStatusPublished,
			},
		},
	}

	// 0 confirmed bookings, but 2 historical cancelled bookings in the database
	bookingRepo := &mockAdminBookingRepo{confirmedCount: 0, totalCount: 2}
	resRepo := &mockAdminReservationRepo{activeCount: 0}

	txRepos := &mockAdminTxRepos{
		eventRepo:       eventRepo,
		bookingRepo:     bookingRepo,
		reservationRepo: resRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := uc.DeleteEvent(context.Background(), uuid.New(), eventID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventCannotBeDeletedWithBookings)
	assert.NotContains(t, eventRepo.deletedIDs, eventID)
}

func TestAdminEventUsecase_DeleteEvent_HasTicketsConflict(t *testing.T) {
	eventID := uuid.New()
	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:     eventID,
				Title:  "Ticketed Event",
				Status: domain.EventStatusPublished,
			},
		},
	}

	bookingRepo := &mockAdminBookingRepo{confirmedCount: 0, totalCount: 0}
	resRepo := &mockAdminReservationRepo{activeCount: 0}
	ticketRepo := &mockAdminTicketRepo{ticketCount: 3}

	txRepos := &mockAdminTxRepos{
		eventRepo:       eventRepo,
		bookingRepo:     bookingRepo,
		reservationRepo: resRepo,
		ticketRepo:      ticketRepo,
	}
	txMgr := &mockTxManager{txRepos: txRepos}

	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := uc.DeleteEvent(context.Background(), uuid.New(), eventID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventCannotBeDeletedWithBookings)
	assert.NotContains(t, eventRepo.deletedIDs, eventID)
}

func TestAdminEventUsecase_UpdateEvent_AgeRestrictionScenarios(t *testing.T) {
	eventID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()

	setupEvent := func(initialAge int) (*AdminEventUsecase, *mockAdminEventRepo) {
		eventRepo := &mockAdminEventRepo{
			events: map[uuid.UUID]*domain.Event{
				eventID: {
					ID:             eventID,
					OrganizerID:    orgID,
					Title:          "Festival",
					Status:         domain.EventStatusDraft,
					AgeRestriction: initialAge,
				},
			},
		}
		txRepos := &mockAdminTxRepos{
			eventRepo: eventRepo,
			organizerRepo: &mockAdminOrganizerRepo{
				organizers: map[uuid.UUID]*domain.Organizer{
					orgID: {ID: orgID, UserID: userID, Status: "ACTIVE"},
				},
			},
			organizerProfileRepo: &mockAdminOrganizerProfileRepo{
				profiles: map[uuid.UUID]*domain.OrganizerProfile{
					orgID: {OrganizerID: orgID, BusinessName: "Org"},
				},
			},
			userRepo: &mockAdminUserRepo{
				users: map[uuid.UUID]*domain.User{
					userID: {ID: userID, Email: "org@example.com"},
				},
			},
			categoryRepo:     &mockAdminCategoryRepo{categories: map[uuid.UUID]*domain.Category{}},
			bookingRepo:      &mockAdminBookingRepo{},
			reservationRepo:  &mockAdminReservationRepo{},
			scheduleRepo:     &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
			settingRepo:      &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
			cancellationRepo: &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
			contactRepo:      &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
			ticketTypeRepo:   &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
			seatLayoutRepo:   &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
			venueRepo:        &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
		}
		txMgr := &mockTxManager{txRepos: txRepos}
		uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
		return uc, eventRepo
	}

	t.Run("omitted age_restriction preserves existing value", func(t *testing.T) {
		uc, repo := setupEvent(18)
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:        eventID,
			Title:          "Renamed Festival",
			AgeRestriction: nil, // omitted
		})
		require.NoError(t, err)
		assert.Equal(t, 18, repo.events[eventID].AgeRestriction)
		assert.Equal(t, "Renamed Festival", repo.events[eventID].Title)
	})

	t.Run("explicitly supplied 0 sets age_restriction to 0", func(t *testing.T) {
		uc, repo := setupEvent(18)
		zeroAge := 0
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:        eventID,
			AgeRestriction: &zeroAge,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, repo.events[eventID].AgeRestriction)
	})

	t.Run("negative age_restriction is rejected", func(t *testing.T) {
		uc, repo := setupEvent(18)
		negAge := -5
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:        eventID,
			AgeRestriction: &negAge,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidEventInput)
		assert.Equal(t, 18, repo.events[eventID].AgeRestriction) // unchanged
	})
}

func TestAdminEventUsecase_UpdateEvent_ContactPartialUpdatesAndClearing(t *testing.T) {
	eventID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()

	initialContact := &domain.EventContact{
		ID:      uuid.New(),
		EventID: eventID,
		Name:    "Alice Organizer",
		Phone:   "+1234567890",
		Email:   "alice@example.com",
	}

	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:          eventID,
				OrganizerID: orgID,
				Title:       "Contact Test Event",
				Status:      domain.EventStatusDraft,
			},
		},
	}
	contactRepo := &mockAdminContactRepo{
		contacts: map[uuid.UUID]*domain.EventContact{
			eventID: initialContact,
		},
	}
	txRepos := &mockAdminTxRepos{
		eventRepo: eventRepo,
		organizerRepo: &mockAdminOrganizerRepo{
			organizers: map[uuid.UUID]*domain.Organizer{
				orgID: {ID: orgID, UserID: userID, Status: "ACTIVE"},
			},
		},
		organizerProfileRepo: &mockAdminOrganizerProfileRepo{
			profiles: map[uuid.UUID]*domain.OrganizerProfile{
				orgID: {OrganizerID: orgID, BusinessName: "Org"},
			},
		},
		userRepo: &mockAdminUserRepo{
			users: map[uuid.UUID]*domain.User{
				userID: {ID: userID, Email: "org@example.com"},
			},
		},
		categoryRepo:     &mockAdminCategoryRepo{categories: map[uuid.UUID]*domain.Category{}},
		bookingRepo:      &mockAdminBookingRepo{},
		reservationRepo:  &mockAdminReservationRepo{},
		scheduleRepo:     &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:      &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo: &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:      contactRepo,
		ticketTypeRepo:   &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:   &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		venueRepo:        &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}
	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Run("updating only contact name preserves existing phone and email", func(t *testing.T) {
		newName := "Alice Smith"
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:     eventID,
			ContactName: &newName,
			// Phone and Email omitted (nil)
		})
		require.NoError(t, err)
		assert.Equal(t, "Alice Smith", contactRepo.contacts[eventID].Name)
		assert.Equal(t, "+1234567890", contactRepo.contacts[eventID].Phone)
		assert.Equal(t, "alice@example.com", contactRepo.contacts[eventID].Email)
	})

	t.Run("updating only contact phone preserves existing name and email", func(t *testing.T) {
		newPhone := "+9876543210"
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:      eventID,
			ContactPhone: &newPhone,
		})
		require.NoError(t, err)
		assert.Equal(t, "Alice Smith", contactRepo.contacts[eventID].Name)
		assert.Equal(t, "+9876543210", contactRepo.contacts[eventID].Phone)
		assert.Equal(t, "alice@example.com", contactRepo.contacts[eventID].Email)
	})

	t.Run("explicitly clearing email with empty string clears email while preserving others", func(t *testing.T) {
		emptyEmail := ""
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:      eventID,
			ContactEmail: &emptyEmail,
		})
		require.NoError(t, err)
		assert.Equal(t, "Alice Smith", contactRepo.contacts[eventID].Name)
		assert.Equal(t, "+9876543210", contactRepo.contacts[eventID].Phone)
		assert.Equal(t, "", contactRepo.contacts[eventID].Email)
	})
}

func TestAdminEventUsecase_UpdateEvent_OptionalMetadataPreserveAndClear(t *testing.T) {
	eventID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()

	eventRepo := &mockAdminEventRepo{
		events: map[uuid.UUID]*domain.Event{
			eventID: {
				ID:                  eventID,
				OrganizerID:         orgID,
				Title:               "Meta Event",
				Status:              domain.EventStatusDraft,
				Highlights:          "Key highlights",
				Rules:               "Strict dress code",
				AttendeeInformation: "Bring ID",
			},
		},
	}
	txRepos := &mockAdminTxRepos{
		eventRepo: eventRepo,
		organizerRepo: &mockAdminOrganizerRepo{
			organizers: map[uuid.UUID]*domain.Organizer{
				orgID: {ID: orgID, UserID: userID, Status: "ACTIVE"},
			},
		},
		organizerProfileRepo: &mockAdminOrganizerProfileRepo{
			profiles: map[uuid.UUID]*domain.OrganizerProfile{
				orgID: {OrganizerID: orgID, BusinessName: "Org"},
			},
		},
		userRepo: &mockAdminUserRepo{
			users: map[uuid.UUID]*domain.User{
				userID: {ID: userID, Email: "org@example.com"},
			},
		},
		categoryRepo:     &mockAdminCategoryRepo{categories: map[uuid.UUID]*domain.Category{}},
		bookingRepo:      &mockAdminBookingRepo{},
		reservationRepo:  &mockAdminReservationRepo{},
		scheduleRepo:     &mockAdminScheduleRepo{schedules: map[uuid.UUID]*domain.EventSchedule{}},
		settingRepo:      &mockAdminSettingRepo{settings: map[uuid.UUID]*domain.EventSetting{}},
		cancellationRepo: &mockAdminCancellationRepo{cancellations: map[uuid.UUID]*domain.EventCancellation{}},
		contactRepo:      &mockAdminContactRepo{contacts: map[uuid.UUID]*domain.EventContact{}},
		ticketTypeRepo:   &mockAdminTicketTypeRepo{types: map[uuid.UUID][]*domain.TicketType{}},
		seatLayoutRepo:   &mockAdminSeatLayoutRepo{layouts: map[uuid.UUID]*domain.SeatLayout{}},
		venueRepo:        &mockAdminVenueRepo{venues: map[uuid.UUID]*domain.Venue{}},
	}
	txMgr := &mockTxManager{txRepos: txRepos}
	uc := NewAdminEventUsecase(txMgr, eventRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Run("omitted metadata fields preserve existing values", func(t *testing.T) {
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID: eventID,
			Title:   "Updated Meta Title",
		})
		require.NoError(t, err)
		assert.Equal(t, "Key highlights", eventRepo.events[eventID].Highlights)
		assert.Equal(t, "Strict dress code", eventRepo.events[eventID].Rules)
		assert.Equal(t, "Bring ID", eventRepo.events[eventID].AttendeeInformation)
	})

	t.Run("empty string explicitly clears metadata field", func(t *testing.T) {
		emptyStr := ""
		_, err := uc.UpdateEvent(context.Background(), uuid.New(), domain.AdminUpdateEventInput{
			EventID:    eventID,
			Highlights: &emptyStr,
		})
		require.NoError(t, err)
		assert.Equal(t, "", eventRepo.events[eventID].Highlights)
		assert.Equal(t, "Strict dress code", eventRepo.events[eventID].Rules)
		assert.Equal(t, "Bring ID", eventRepo.events[eventID].AttendeeInformation)
	})
}

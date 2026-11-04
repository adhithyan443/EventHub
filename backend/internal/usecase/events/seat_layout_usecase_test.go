package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations for SeatLayoutUsecase Tests ---

type mockLayoutTxManager struct {
	txRepos *mockLayoutTxRepos
}

func (m *mockLayoutTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

type mockLayoutTxRepos struct {
	domain.TransactionRepositories
	organizerRepo   domain.OrganizerRepository
	eventRepo       domain.EventRepository
	settingRepo     domain.EventSettingRepository
	ticketTypeRepo  domain.TicketTypeRepository
	seatLayoutRepo  domain.SeatLayoutRepository
	seatSectionRepo domain.SeatSectionRepository
	seatRowRepo     domain.SeatRowRepository
	seatRepo        domain.SeatRepository
	seatColumnRepo  domain.SeatColumnRepository
}

func (m *mockLayoutTxRepos) OrganizerRepository() domain.OrganizerRepository { return m.organizerRepo }
func (m *mockLayoutTxRepos) EventRepository() domain.EventRepository { return m.eventRepo }
func (m *mockLayoutTxRepos) EventSettingRepository() domain.EventSettingRepository { return m.settingRepo }
func (m *mockLayoutTxRepos) TicketTypeRepository() domain.TicketTypeRepository { return m.ticketTypeRepo }
func (m *mockLayoutTxRepos) SeatLayoutRepository() domain.SeatLayoutRepository { return m.seatLayoutRepo }
func (m *mockLayoutTxRepos) SeatSectionRepository() domain.SeatSectionRepository { return m.seatSectionRepo }
func (m *mockLayoutTxRepos) SeatRowRepository() domain.SeatRowRepository { return m.seatRowRepo }
func (m *mockLayoutTxRepos) SeatRepository() domain.SeatRepository { return m.seatRepo }
func (m *mockLayoutTxRepos) SeatColumnRepository() domain.SeatColumnRepository { return m.seatColumnRepo }

type mockSLOrganizerRepo struct {
	domain.OrganizerRepository
	organizer *domain.Organizer
}

func (m *mockSLOrganizerRepo) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.organizer != nil && m.organizer.UserID == userID {
		return m.organizer, nil
	}
	return nil, errors.New("organizer not found")
}

type mockSLEventRepo struct {
	domain.EventRepository
	event                   *domain.Event
	findByIDForUpdateCalled bool
}

func (m *mockSLEventRepo) FindByID(id uuid.UUID) (*domain.Event, error) {
	if m.event != nil && m.event.ID == id {
		return m.event, nil
	}
	return nil, domain.ErrEventNotFound
}

func (m *mockSLEventRepo) FindByIDForUpdate(id uuid.UUID) (*domain.Event, error) {
	m.findByIDForUpdateCalled = true
	return m.FindByID(id)
}

type mockSLEventSettingRepo struct {
	domain.EventSettingRepository
	setting *domain.EventSetting
}

func (m *mockSLEventSettingRepo) FindByEventID(eventID uuid.UUID) (*domain.EventSetting, error) {
	if m.setting != nil && m.setting.EventID == eventID {
		return m.setting, nil
	}
	return nil, errors.New("setting not found")
}

type mockSLTicketTypeRepo struct {
	domain.TicketTypeRepository
	ticketTypes map[uuid.UUID]*domain.TicketType
}

func (m *mockSLTicketTypeRepo) Create(ctx context.Context, tt *domain.TicketType) error {
	if m.ticketTypes == nil {
		m.ticketTypes = make(map[uuid.UUID]*domain.TicketType)
	}
	m.ticketTypes[tt.ID] = tt
	return nil
}
func (m *mockSLTicketTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.TicketType, error) {
	if tt, ok := m.ticketTypes[id]; ok {
		return tt, nil
	}
	return nil, errors.New("ticket type not found")
}
func (m *mockSLTicketTypeRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.TicketType, error) {
	var list []*domain.TicketType
	for _, tt := range m.ticketTypes {
		if tt.EventID == eventID {
			list = append(list, tt)
		}
	}
	return list, nil
}
func (m *mockSLTicketTypeRepo) Update(ctx context.Context, tt *domain.TicketType) error {
	m.ticketTypes[tt.ID] = tt
	return nil
}
func (m *mockSLTicketTypeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.ticketTypes, id)
	return nil
}
func (m *mockSLTicketTypeRepo) DeleteByEventID(ctx context.Context, eventID uuid.UUID) error {
	for id, tt := range m.ticketTypes {
		if tt.EventID == eventID {
			delete(m.ticketTypes, id)
		}
	}
	return nil
}

type mockSLSeatLayoutRepo struct {
	domain.SeatLayoutRepository
	layout *domain.SeatLayout
}

func (m *mockSLSeatLayoutRepo) Create(layout *domain.SeatLayout) error {
	m.layout = layout
	return nil
}
func (m *mockSLSeatLayoutRepo) FindByID(id uuid.UUID) (*domain.SeatLayout, error) {
	if m.layout != nil && m.layout.ID == id {
		return m.layout, nil
	}
	return nil, domain.ErrSeatLayoutNotFound
}
func (m *mockSLSeatLayoutRepo) FindByEventID(eventID uuid.UUID) (*domain.SeatLayout, error) {
	if m.layout != nil && m.layout.EventID == eventID {
		return m.layout, nil
	}
	return nil, domain.ErrSeatLayoutNotFound
}
func (m *mockSLSeatLayoutRepo) Update(layout *domain.SeatLayout) error {
	m.layout = layout
	return nil
}

type mockSLSeatSectionRepo struct {
	domain.SeatSectionRepository
	sections map[uuid.UUID]domain.SeatSection
}

func (m *mockSLSeatSectionRepo) Create(sec *domain.SeatSection) error {
	if m.sections == nil {
		m.sections = make(map[uuid.UUID]domain.SeatSection)
	}
	m.sections[sec.ID] = *sec
	return nil
}
func (m *mockSLSeatSectionRepo) FindBySeatLayoutID(layoutID uuid.UUID) ([]domain.SeatSection, error) {
	var list []domain.SeatSection
	for _, s := range m.sections {
		if s.SeatLayoutID == layoutID {
			list = append(list, s)
		}
	}
	return list, nil
}
func (m *mockSLSeatSectionRepo) Update(sec *domain.SeatSection) error {
	m.sections[sec.ID] = *sec
	return nil
}
func (m *mockSLSeatSectionRepo) Delete(id uuid.UUID) error {
	delete(m.sections, id)
	return nil
}

type mockSLSeatRowRepo struct {
	domain.SeatRowRepository
	rows map[uuid.UUID]domain.SeatRow
}

func (m *mockSLSeatRowRepo) Create(row *domain.SeatRow) error {
	if m.rows == nil {
		m.rows = make(map[uuid.UUID]domain.SeatRow)
	}
	m.rows[row.ID] = *row
	return nil
}
func (m *mockSLSeatRowRepo) FindBySectionID(secID uuid.UUID) ([]domain.SeatRow, error) {
	var list []domain.SeatRow
	for _, r := range m.rows {
		if r.SectionID == secID {
			list = append(list, r)
		}
	}
	return list, nil
}
func (m *mockSLSeatRowRepo) Update(row *domain.SeatRow) error {
	m.rows[row.ID] = *row
	return nil
}
func (m *mockSLSeatRowRepo) Delete(id uuid.UUID) error {
	delete(m.rows, id)
	return nil
}

type mockSLSeatColumnRepo struct {
	domain.SeatColumnRepository
	columns map[uuid.UUID]domain.SeatColumn
}

func (m *mockSLSeatColumnRepo) Create(col *domain.SeatColumn) error {
	if m.columns == nil {
		m.columns = make(map[uuid.UUID]domain.SeatColumn)
	}
	m.columns[col.ID] = *col
	return nil
}
func (m *mockSLSeatColumnRepo) CreateBatch(cols []domain.SeatColumn) error {
	for _, c := range cols {
		_ = m.Create(&c)
	}
	return nil
}
func (m *mockSLSeatColumnRepo) FindBySeatLayoutID(layoutID uuid.UUID) ([]domain.SeatColumn, error) {
	var list []domain.SeatColumn
	for _, c := range m.columns {
		if c.SeatLayoutID == layoutID {
			list = append(list, c)
		}
	}
	return list, nil
}
func (m *mockSLSeatColumnRepo) FindByID(id uuid.UUID) (*domain.SeatColumn, error) {
	if c, ok := m.columns[id]; ok {
		return &c, nil
	}
	return nil, errors.New("seat column not found")
}
func (m *mockSLSeatColumnRepo) Update(col *domain.SeatColumn) error {
	m.columns[col.ID] = *col
	return nil
}
func (m *mockSLSeatColumnRepo) Delete(id uuid.UUID) error {
	delete(m.columns, id)
	return nil
}
func (m *mockSLSeatColumnRepo) DeleteBySeatLayoutID(layoutID uuid.UUID) error {
	for id, c := range m.columns {
		if c.SeatLayoutID == layoutID {
			delete(m.columns, id)
		}
	}
	return nil
}

type mockSLSeatRepo struct {
	domain.SeatRepository
	seats                     map[uuid.UUID]domain.Seat
	bookedOrReservedSeatIDs   map[uuid.UUID]bool
	bookedOrReservedColumnIDs map[uuid.UUID]bool
	bookedCountByLayoutID     int64
}

func (m *mockSLSeatRepo) Create(seat *domain.Seat) error {
	if m.seats == nil {
		m.seats = make(map[uuid.UUID]domain.Seat)
	}
	m.seats[seat.ID] = *seat
	return nil
}
func (m *mockSLSeatRepo) FindByRowID(rowID uuid.UUID) ([]domain.Seat, error) {
	var list []domain.Seat
	for _, s := range m.seats {
		if s.RowID == rowID {
			list = append(list, s)
		}
	}
	return list, nil
}
func (m *mockSLSeatRepo) Update(seat *domain.Seat) error {
	m.seats[seat.ID] = *seat
	return nil
}
func (m *mockSLSeatRepo) Delete(id uuid.UUID) error {
	delete(m.seats, id)
	return nil
}
func (m *mockSLSeatRepo) FindByID(id uuid.UUID) (*domain.Seat, error) {
	if s, ok := m.seats[id]; ok {
		return &s, nil
	}
	return nil, errors.New("seat not found")
}
func (m *mockSLSeatRepo) FindByColumnID(ctx context.Context, colID uuid.UUID) ([]domain.Seat, error) {
	var list []domain.Seat
	for _, s := range m.seats {
		if s.ColumnID != nil && *s.ColumnID == colID {
			list = append(list, s)
		}
	}
	return list, nil
}
func (m *mockSLSeatRepo) CountBookedByLayoutID(layoutID uuid.UUID) (int64, error) {
	return m.bookedCountByLayoutID, nil
}
func (m *mockSLSeatRepo) CountBookedOrReservedByColumnID(ctx context.Context, columnID uuid.UUID) (int64, error) {
	if m.bookedOrReservedColumnIDs != nil && m.bookedOrReservedColumnIDs[columnID] {
		return 1, nil
	}
	return 0, nil
}
func (m *mockSLSeatRepo) CountBookedOrReservedBySeatIDs(ctx context.Context, seatIDs []uuid.UUID) (int64, error) {
	var count int64
	for _, id := range seatIDs {
		if m.bookedOrReservedSeatIDs != nil && m.bookedOrReservedSeatIDs[id] {
			count++
		}
	}
	return count, nil
}

// --- Tests ---

func TestGenerateDefaultAlphabeticLabel(t *testing.T) {
	testCases := []struct {
		index    int
		expected string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, GenerateDefaultAlphabeticLabel(tc.index))
	}
}

func setupSeatLayoutUsecaseTest() (*SeatLayoutUsecase, *mockLayoutTxRepos, uuid.UUID, uuid.UUID) {
	userID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()

	organizer := &domain.Organizer{
		ID:     organizerID,
		UserID: userID,
		Status: "ACTIVE",
	}

	event := &domain.Event{
		ID:          eventID,
		OrganizerID: organizerID,
		EventType:   "PHYSICAL",
		Status:      "DRAFT",
	}

	setting := &domain.EventSetting{
		ID:             uuid.New(),
		EventID:        eventID,
		SeatLayoutType: SeatLayoutTypeSeated,
	}

	txRepos := &mockLayoutTxRepos{
		organizerRepo:   &mockSLOrganizerRepo{organizer: organizer},
		eventRepo:       &mockSLEventRepo{event: event},
		settingRepo:     &mockSLEventSettingRepo{setting: setting},
		ticketTypeRepo:  &mockSLTicketTypeRepo{ticketTypes: make(map[uuid.UUID]*domain.TicketType)},
		seatLayoutRepo:  &mockSLSeatLayoutRepo{},
		seatSectionRepo: &mockSLSeatSectionRepo{sections: make(map[uuid.UUID]domain.SeatSection)},
		seatRowRepo:     &mockSLSeatRowRepo{rows: make(map[uuid.UUID]domain.SeatRow)},
		seatRepo:        &mockSLSeatRepo{seats: make(map[uuid.UUID]domain.Seat)},
		seatColumnRepo:  &mockSLSeatColumnRepo{columns: make(map[uuid.UUID]domain.SeatColumn)},
	}

	txManager := &mockLayoutTxManager{txRepos: txRepos}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	usecase := NewSeatLayoutUsecase(
		txManager,
		logger,
	)

	return usecase, txRepos, userID, eventID
}

func TestCreateSeatLayout_WithColumnsAndGaps(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()
	col3ID := uuid.New()

	input := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
			{ID: &col2ID, Label: "Col-2", OrderIndex: 1},
			{ID: &col3ID, Label: "Col-3", OrderIndex: 2},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2, // 2 seats, gap at column 2 (Col-2)
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1, Status: "AVAILABLE"},
							{ColumnID: &col3ID, GridCol: 2, SeatNumber: 2, Status: "AVAILABLE"},
						},
					},
				},
			},
		},
	}

	output, err := usecase.CreateSeatLayout(input)
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Equal(t, 3, len(output.Columns))
	assert.Equal(t, 1, len(output.Sections))
	assert.Equal(t, 1, len(output.Rows))
	assert.Equal(t, 2, len(output.Seats))

	// Verify column references
	assert.Equal(t, col1ID, *output.Seats[0].ColumnID)
	assert.Equal(t, 0, output.Seats[0].GridCol)
	assert.Equal(t, col3ID, *output.Seats[1].ColumnID)
	assert.Equal(t, 2, output.Seats[1].GridCol)

	// Verify columns stored in repo
	storedCols, err := txRepos.seatColumnRepo.FindBySeatLayoutID(output.Layout.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(storedCols))
}

func TestCreateSeatLayout_DefaultColumnsGenerated(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	input := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Sections: []SeatSectionInput{
			{
				Name:  "Standard",
				Price: decimal.NewFromInt(50),
				Rows: []SeatRowInput{
					{RowName: "A", Seats: 4},
				},
			},
		},
	}

	output, err := usecase.CreateSeatLayout(input)
	require.NoError(t, err)
	assert.Equal(t, 4, len(output.Columns))
	assert.Equal(t, "A", output.Columns[0].Label)
	assert.Equal(t, "B", output.Columns[1].Label)
	assert.Equal(t, "C", output.Columns[2].Label)
	assert.Equal(t, "D", output.Columns[3].Label)
}

func TestUpdateSeatLayout_AddRenameReorderColumns(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	// Initial create with 2 columns
	col1ID := uuid.New()
	col2ID := uuid.New()
	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
			{ID: &col2ID, Label: "Col-2", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Now update: reorder Col-2 to index 0, rename Col-1 to Col-Renamed at index 1, add new Col-3 at index 2
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall Updated",
		Columns: []UpdateSeatColumnInput{
			{ID: &col2ID, Label: "Col-2", OrderIndex: 0},
			{ID: &col1ID, Label: "Col-Renamed", OrderIndex: 1},
			{Label: "Col-3", OrderIndex: 2}, // new column without ID
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 1, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 0, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	updateOutput, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.NoError(t, err)
	assert.Equal(t, 3, len(updateOutput.Columns))

	// Verify columns in repo
	cols, err := txRepos.seatColumnRepo.FindBySeatLayoutID(createOutput.Layout.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(cols))
}

func TestUpdateSeatLayout_RejectDuplicateColumnLabel(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{Label: "DUPLICATE", OrderIndex: 0},
			{Label: "DUPLICATE", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				Name:  "Standard",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{RowName: "A", Seats: 1},
				},
			},
		},
	}

	_, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatColumnDuplicateLabel)
}

func TestUpdateSeatLayout_RejectDuplicateColumnOrder(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{Label: "Col-1", OrderIndex: 0},
			{Label: "Col-2", OrderIndex: 0},
		},
		Sections: []UpdateSeatSectionInput{
			{
				Name:  "Standard",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{RowName: "A", Seats: 1},
				},
			},
		},
	}

	_, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatColumnDuplicateOrder)
}

func TestUpdateSeatLayout_RejectRemovalOfBookedColumn(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()
	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
			{ID: &col2ID, Label: "Col-2", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate booked seat in col2
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedOrReservedColumnIDs = map[uuid.UUID]bool{col2ID: true}

	// Update removes col2 (only sends col1)
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   1,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutBookedSeatCannotBeRemoved)
}

func TestUpdateSeatLayout_RejectRemovalOfBookedSeat(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()
	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
			{ID: &col2ID, Label: "Col-2", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate seat 2 is booked
	seat2ID := createOutput.Seats[1].ID
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedOrReservedSeatIDs = map[uuid.UUID]bool{seat2ID: true}

	// Update tries to remove seat 2
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "Col-1", OrderIndex: 0},
			{ID: &col2ID, Label: "Col-2", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   1,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutBookedSeatCannotBeRemoved)
}

func TestCreateSeatLayout_RejectDuplicateGridColZero(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	input := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 0, SeatNumber: 2}, // Duplicate GridCol 0
						},
					},
				},
			},
		},
	}

	_, err := usecase.CreateSeatLayout(input)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSeatRow)
}

func TestCreateSeatLayout_RejectNegativeGridCol(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()

	input := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   1,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: -1, SeatNumber: 1}, // Negative GridCol
						},
					},
				},
			},
		},
	}

	_, err := usecase.CreateSeatLayout(input)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSeatRow)
}

func TestUpdateSeatLayout_RejectDuplicateGridColZero(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 0, SeatNumber: 2}, // Duplicate GridCol 0
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSeatRow)
}

func TestUpdateSeatLayout_RejectNegativeGridCol(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   1,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   1,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: -1, SeatNumber: 1}, // Negative
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSeatRow)
}

func TestUpdateSeatLayout_RejectColumnReorderingWhenBookingsExist(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate that bookings exist on this layout
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedCountByLayoutID = 2

	// Organizer attempts to reorder columns (swap OrderIndex: Col A -> 1, Col B -> 0)
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 1},
			{ID: &col2ID, Label: "B", OrderIndex: 0},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutStructuralChangesNotAllowed)
}

func TestUpdateSeatLayout_AllowColumnLabelRenameWhenBookingsExist(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate that bookings exist on this layout
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedCountByLayoutID = 2

	// Organizer renames column label only (Col A -> "Alpha"), preserving ID and OrderIndex
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "Alpha", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	output, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.NoError(t, err)
	assert.Equal(t, "Alpha", output.Columns[0].Label)
}

func TestUpdateSeatLayout_PreserveGridColZero(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	output, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.NoError(t, err)
	require.Len(t, output.Seats, 2)

	// Verify Seat 1 preserves GridCol 0 and is not overwritten with SeatNumber (1)
	assert.Equal(t, 0, output.Seats[0].GridCol)
	assert.Equal(t, 1, output.Seats[0].SeatNumber)

	// Verify Seat 2 preserves GridCol 1
	assert.Equal(t, 1, output.Seats[1].GridCol)
	assert.Equal(t, 2, output.Seats[1].SeatNumber)
}

func TestUpdateSeatLayout_AdjacentSeatsRetainDistinctZeroAndOne(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Update existing seats and verify they do not collapse into the same column
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	output, err := usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.NoError(t, err)
	require.Len(t, output.Seats, 2)

	assert.NotEqual(t, output.Seats[0].GridCol, output.Seats[1].GridCol, "Adjacent seats must have distinct GridCol positions")
	assert.Equal(t, 0, output.Seats[0].GridCol)
	assert.Equal(t, 1, output.Seats[1].GridCol)
}

func TestUpdateSeatLayout_MissingGridColBehavior(t *testing.T) {
	usecase, _, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Scenario 1: Multiple seats in row with missing/omitted GridCol (both default to 0 in Go)
	// Must be rejected because two seats cannot occupy the same column 0
	multiSeatMissingColInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, SeatNumber: 1}, // GridCol omitted (defaults to 0)
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, SeatNumber: 2}, // GridCol omitted (defaults to 0)
						},
					},
				},
			},
		},
	}
	_, err = usecase.UpdateSeatLayout(context.Background(), multiSeatMissingColInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSeatRow)

	// Scenario 2: Single seat in row with missing/omitted GridCol (defaults to 0)
	// Succeeds and takes valid column 0
	singleSeatMissingColInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   1,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, SeatNumber: 1}, // GridCol omitted -> defaults to 0
						},
					},
				},
			},
		},
	}
	output, err := usecase.UpdateSeatLayout(context.Background(), singleSeatMissingColInput)
	require.NoError(t, err)
	require.Len(t, output.Seats, 1)
	assert.Equal(t, 0, output.Seats[0].GridCol)
}

func TestUpdateSeatLayout_AcquiresEventLockForUpdate(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   1,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Reset event repo lock flag
	eventRepo := txRepos.eventRepo.(*mockSLEventRepo)
	eventRepo.findByIDForUpdateCalled = false

	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "General",
				Price: decimal.NewFromInt(50),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   1,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.NoError(t, err)

	// Verify FindByIDForUpdate was explicitly called to acquire the row-level lock
	assert.True(t, eventRepo.findByIDForUpdateCalled, "UpdateSeatLayout must acquire exclusive event row lock with FindByIDForUpdate")
}

func TestUpdateSeatLayout_RejectColumnReplacementWhenBookingsExist(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate that bookings exist on this layout
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedCountByLayoutID = 1

	// Organizer attempts to replace Col 2 with a brand new column ID (same total count = 2)
	newCol2ID := uuid.New()
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &newCol2ID, Label: "B-New", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &newCol2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutStructuralChangesNotAllowed)

	// Verify no destructive repository operations occurred
	colRepo := txRepos.seatColumnRepo.(*mockSLSeatColumnRepo)
	assert.Len(t, colRepo.columns, 2, "Column count in repo must remain intact")
	_, existsOldCol := colRepo.columns[col2ID]
	assert.True(t, existsOldCol, "Original column must not be deleted")
	_, existsNewCol := colRepo.columns[newCol2ID]
	assert.False(t, existsNewCol, "Replaced column must not be created")
}

func TestUpdateSeatLayout_RejectBookedSeatGridColChange(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()
	col3ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
			{ID: &col3ID, Label: "C", OrderIndex: 2},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate that bookings exist on this layout
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedCountByLayoutID = 1

	// Organizer attempts to shift Seat 2's GridCol from 1 to 2
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
			{ID: &col3ID, Label: "C", OrderIndex: 2},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ID: &createOutput.Seats[1].ID, ColumnID: &col3ID, GridCol: 2, SeatNumber: 2}, // Shifted GridCol 1 -> 2
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutStructuralChangesNotAllowed)

	// Verify no destructive repository operations occurred on seats
	seat1, err1 := txRepos.seatRepo.FindByID(createOutput.Seats[0].ID)
	require.NoError(t, err1)
	assert.Equal(t, 0, seat1.GridCol, "Seat 1 position must be preserved")

	seat2, err2 := txRepos.seatRepo.FindByID(createOutput.Seats[1].ID)
	require.NoError(t, err2)
	assert.Equal(t, 1, seat2.GridCol, "Seat 2 position must be preserved at original GridCol 1")
}

func TestUpdateSeatLayout_RejectBookedSeatColumnIDReassignment(t *testing.T) {
	usecase, txRepos, userID, eventID := setupSeatLayoutUsecaseTest()

	col1ID := uuid.New()
	col2ID := uuid.New()

	createInput := CreateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []SeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []SeatSectionInput{
			{
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []SeatRowInput{
					{
						RowName: "A",
						Seats:   2,
						SeatItems: []SeatItemInput{
							{ColumnID: &col1ID, GridCol: 0, SeatNumber: 1},
							{ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}
	createOutput, err := usecase.CreateSeatLayout(createInput)
	require.NoError(t, err)

	// Simulate that bookings exist on this layout
	mockSeats := txRepos.seatRepo.(*mockSLSeatRepo)
	mockSeats.bookedCountByLayoutID = 1

	// Organizer attempts to reassign Seat 1's ColumnID from col1ID to col2ID (or nil)
	updateInput := UpdateSeatLayoutInput{
		UserID:     userID,
		EventID:    eventID,
		LayoutName: "Main Hall",
		Columns: []UpdateSeatColumnInput{
			{ID: &col1ID, Label: "A", OrderIndex: 0},
			{ID: &col2ID, Label: "B", OrderIndex: 1},
		},
		Sections: []UpdateSeatSectionInput{
			{
				ID:    &createOutput.Sections[0].ID,
				Name:  "VIP",
				Price: decimal.NewFromInt(100),
				Rows: []UpdateSeatRowInput{
					{
						ID:      &createOutput.Rows[0].ID,
						RowName: "A",
						Seats:   2,
						SeatItems: []UpdateSeatItemInput{
							{ID: &createOutput.Seats[0].ID, ColumnID: &col2ID, GridCol: 0, SeatNumber: 1}, // ColumnID reassigned col1ID -> col2ID
							{ID: &createOutput.Seats[1].ID, ColumnID: &col2ID, GridCol: 1, SeatNumber: 2},
						},
					},
				},
			},
		},
	}

	_, err = usecase.UpdateSeatLayout(context.Background(), updateInput)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSeatLayoutStructuralChangesNotAllowed)

	// Verify no destructive repository operations occurred on seats
	seat1, err1 := txRepos.seatRepo.FindByID(createOutput.Seats[0].ID)
	require.NoError(t, err1)
	require.NotNil(t, seat1.ColumnID)
	assert.Equal(t, col1ID, *seat1.ColumnID, "Seat 1 column assignment must remain intact")
}

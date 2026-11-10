package events

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrSeatLayoutEventNotFound               = errors.New("event not found")
	ErrInvalidSeatLayout                     = errors.New("invalid seat layout")
	ErrInvalidSeatSection                    = errors.New("invalid seat section")
	ErrInvalidSeatRow                        = errors.New("invalid seat row")
	ErrInvalidSeatColumn                     = errors.New("invalid seat column")
	ErrSeatColumnDuplicateLabel              = errors.New("seat column label must be unique")
	ErrSeatColumnDuplicateOrder              = errors.New("seat column order must be unique")
	ErrSeatLayoutDraftOnly                   = errors.New("seat layout can only be modified for draft events")
	ErrSeatLayoutSeatedOnly                  = errors.New("seat layout is only supported for seated events")
	ErrSeatLayoutStructuralChangesNotAllowed = errors.New("structural changes to seat layout are not allowed once bookings exist")
	ErrSeatLayoutDuplicateID                 = errors.New("duplicate or conflicting seat layout ID")
	ErrSeatLayoutInvalidStatus               = errors.New("invalid seat status")
	ErrSeatLayoutBookedSeatCannotBeRemoved   = errors.New("cannot remove or modify columns or seats that have active reservations or bookings")
)

type BookingGuard interface {
	HasBookings(ctx context.Context, eventID uuid.UUID) (bool, error)
}

type CreateSeatLayoutInput struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	Columns    []SeatColumnInput
	Sections   []SeatSectionInput
}

type SeatColumnInput struct {
	ID         *uuid.UUID
	Label      string
	OrderIndex int
}

type SeatSectionInput struct {
	Name  string
	Price decimal.Decimal
	Rows  []SeatRowInput
}

type SeatRowInput struct {
	RowName   string
	Seats     int
	SeatItems []SeatItemInput
}

type SeatItemInput struct {
	ID         *uuid.UUID
	ColumnID   *uuid.UUID
	GridCol    int
	SeatNumber int
	Status     string
}

type CreateSeatLayoutOutput struct {
	Layout   *domain.SeatLayout
	Columns  []domain.SeatColumn
	Sections []domain.SeatSection
	Rows     []domain.SeatRow
	Seats    []domain.Seat
}

type UpdateSeatLayoutInput struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	Columns    []UpdateSeatColumnInput
	Sections   []UpdateSeatSectionInput
}

type UpdateSeatColumnInput struct {
	ID         *uuid.UUID
	Label      string
	OrderIndex int
}

type UpdateSeatSectionInput struct {
	ID    *uuid.UUID
	Name  string
	Price decimal.Decimal
	Rows  []UpdateSeatRowInput
}

type UpdateSeatRowInput struct {
	ID        *uuid.UUID
	RowName   string
	Seats     int
	SeatItems []UpdateSeatItemInput
}

type UpdateSeatItemInput struct {
	ID         *uuid.UUID
	ColumnID   *uuid.UUID
	GridCol    int
	SeatNumber int
	Status     string
}

type UpdateSeatLayoutOutput struct {
	Layout   *domain.SeatLayout
	Columns  []domain.SeatColumn
	Sections []domain.SeatSection
	Rows     []domain.SeatRow
	Seats    []domain.Seat
}

type SeatLayoutUsecase struct {
	transactionManager domain.TransactionManager
	bookingGuard       BookingGuard
	logger             *slog.Logger
}

func NewSeatLayoutUsecase(
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *SeatLayoutUsecase {
	return &SeatLayoutUsecase{
		transactionManager: transactionManager,
		logger:             logger,
	}
}

func (u *SeatLayoutUsecase) SetBookingGuard(guard BookingGuard) {
	u.bookingGuard = guard
}

// GenerateDefaultAlphabeticLabel returns A, B, C, ... Z, AA, AB, etc. for 1-based index.
func GenerateDefaultAlphabeticLabel(orderIndex int) string {
	if orderIndex < 0 {
		return "A"
	}
	result := ""
	curr := orderIndex + 1
	for curr > 0 {
		curr--
		rem := curr % 26
		result = string(rune('A'+rem)) + result
		curr /= 26
	}
	return result
}

func (u *SeatLayoutUsecase) CreateSeatLayout(
	input CreateSeatLayoutInput,
) (*CreateSeatLayoutOutput, error) {
	if err := validateCreateSeatLayoutInput(input); err != nil {
		u.logger.Warn(
			"seat_layout_validation_failed",
			"event_id", input.EventID,
			"user_id", input.UserID,
			"error", err,
		)
		return nil, err
	}

	var output CreateSeatLayoutOutput

	err := u.transactionManager.WithinTransaction(
		func(repos domain.TransactionRepositories) error {
			// Find the organizer associated with the authenticated user.
			organizer, err := repos.OrganizerRepository().FindByUserID(input.UserID)
			if err != nil {
				u.logger.Warn(
					"seat_layout_organizer_lookup_failed",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"error", err,
				)
				return ErrUnauthorizedOrganizer
			}

			// The organizer itself must be active.
			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"seat_layout_inactive_organizer",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"organizer_status", organizer.Status,
				)
				return ErrUnauthorizedOrganizer
			}

			// Find the event.
			event, err := repos.EventRepository().FindByIDForUpdate(input.EventID)
			if err != nil {
				return err
			}

			// Verify event ownership.
			if event.OrganizerID != organizer.ID {
				u.logger.Warn(
					"seat_layout_event_ownership_failed",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"organizer_id", organizer.ID,
					"event_organizer_id", event.OrganizerID,
				)
				return ErrUnauthorizedOrganizer
			}

			// Seat layouts are only valid for physical or hybrid events.
			if event.EventType != domain.EventTypePhysical &&
				event.EventType != domain.EventTypeHybrid {
				u.logger.Warn(
					"seat_layout_invalid_event_type",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"event_type", event.EventType,
				)
				return ErrInvalidSeatLayout
			}

			// Fetch event settings.
			setting, err := repos.EventSettingRepository().FindByEventID(input.EventID)
			if err != nil {
				return err
			}

			// Only SEATED events require a seat layout.
			if setting.SeatLayoutType != SeatLayoutTypeSeated {
				u.logger.Warn(
					"seat_layout_invalid_layout_type",
					"event_id", input.EventID,
					"user_id", input.UserID,
					"seat_layout_type", setting.SeatLayoutType,
				)
				return ErrInvalidSeatLayout
			}

			// An event can have only one seat layout.
			if _, err := repos.SeatLayoutRepository().FindByEventID(input.EventID); err == nil {
				u.logger.Warn(
					"seat_layout_already_exists",
					"event_id", input.EventID,
					"user_id", input.UserID,
				)
				return ErrInvalidSeatLayout
			} else if !errors.Is(err, domain.ErrSeatLayoutNotFound) {
				return err
			}

			res, err := createSeatLayout(repos, input.EventID, input)
			if err != nil {
				return err
			}
			output = *res
			return nil
		},
	)

	if err != nil {
		u.logger.Error(
			"seat_layout_creation_failed",
			"event_id", input.EventID,
			"user_id", input.UserID,
			"error", err,
		)
		return nil, err
	}

	u.logger.Info(
		"seat_layout_created",
		"event_id", input.EventID,
		"user_id", input.UserID,
		"seat_layout_id", output.Layout.ID,
		"column_count", len(output.Columns),
		"section_count", len(output.Sections),
		"row_count", len(output.Rows),
		"seat_count", len(output.Seats),
	)

	return &output, nil
}

func createSeatLayout(
	repos domain.TransactionRepositories,
	eventID uuid.UUID,
	input CreateSeatLayoutInput,
) (*CreateSeatLayoutOutput, error) {
	output := &CreateSeatLayoutOutput{}

	layout := &domain.SeatLayout{
		ID:         uuid.New(),
		EventID:    eventID,
		LayoutName: strings.TrimSpace(input.LayoutName),
	}

	if err := repos.SeatLayoutRepository().Create(layout); err != nil {
		return nil, err
	}

	output.Layout = layout

	// Determine layout-wide columns
	columnList := make([]domain.SeatColumn, 0)
	colByID := make(map[uuid.UUID]domain.SeatColumn)
	if len(input.Columns) > 0 {
		for idx, colIn := range input.Columns {
			label := strings.TrimSpace(colIn.Label)
			if label == "" {
				label = GenerateDefaultAlphabeticLabel(idx)
			}
			orderIdx := colIn.OrderIndex
			if orderIdx < 0 {
				orderIdx = idx
			}

			colID := uuid.New()
			if colIn.ID != nil && *colIn.ID != uuid.Nil {
				colID = *colIn.ID
			}

			col := domain.SeatColumn{
				ID:           colID,
				SeatLayoutID: layout.ID,
				Label:        label,
				OrderIndex:   orderIdx,
			}
			if err := repos.SeatColumnRepository().Create(&col); err != nil {
				return nil, err
			}
			columnList = append(columnList, col)
			colByID[col.ID] = col
		}
	} else {
		// Auto-generate default alphabetic columns based on max seats across rows
		maxCols := 0
		for _, sec := range input.Sections {
			for _, row := range sec.Rows {
				if len(row.SeatItems) > 0 {
					for _, item := range row.SeatItems {
						if item.GridCol+1 > maxCols {
							maxCols = item.GridCol + 1
						}
						if item.SeatNumber > maxCols {
							maxCols = item.SeatNumber
						}
					}
				} else if row.Seats > maxCols {
					maxCols = row.Seats
				}
			}
		}

		for c := 0; c < maxCols; c++ {
			col := domain.SeatColumn{
				ID:           uuid.New(),
				SeatLayoutID: layout.ID,
				Label:        GenerateDefaultAlphabeticLabel(c),
				OrderIndex:   c,
			}
			if err := repos.SeatColumnRepository().Create(&col); err != nil {
				return nil, err
			}
			columnList = append(columnList, col)
			colByID[col.ID] = col
		}
	}

	output.Columns = columnList

	// Map columns by order index for rapid coordinate matching
	colByOrder := make(map[int]domain.SeatColumn)
	for _, c := range columnList {
		colByOrder[c.OrderIndex] = c
	}

	for _, sectionInput := range input.Sections {
		section := &domain.SeatSection{
			ID:           uuid.New(),
			SeatLayoutID: layout.ID,
			Name:         strings.TrimSpace(sectionInput.Name),
			Price:        sectionInput.Price,
		}

		if err := repos.SeatSectionRepository().Create(section); err != nil {
			return nil, err
		}

		output.Sections = append(output.Sections, *section)

		for _, rowInput := range sectionInput.Rows {
			row := &domain.SeatRow{
				ID:        uuid.New(),
				SectionID: section.ID,
				RowName:   strings.TrimSpace(rowInput.RowName),
			}

			if err := repos.SeatRowRepository().Create(row); err != nil {
				return nil, err
			}

			output.Rows = append(output.Rows, *row)

			if len(rowInput.SeatItems) > 0 {
				for idx, item := range rowInput.SeatItems {
					gridCol := item.GridCol
					if gridCol < 0 {
						gridCol = idx
					}

					var colID *uuid.UUID
					if item.ColumnID != nil && *item.ColumnID != uuid.Nil {
						if _, ok := colByID[*item.ColumnID]; ok {
							colIDCopy := *item.ColumnID
							colID = &colIDCopy
						}
					}
					if colID == nil {
						if c, ok := colByOrder[gridCol]; ok {
							colIDCopy := c.ID
							colID = &colIDCopy
						} else if len(columnList) > 0 {
							colIDCopy := columnList[0].ID
							colID = &colIDCopy
						}
					}

					status := domain.SeatStatusAvailable
					if item.Status != "" {
						status = strings.ToUpper(item.Status)
					}

					seat := &domain.Seat{
						ID:         uuid.New(),
						RowID:      row.ID,
						ColumnID:   colID,
						GridCol:    gridCol,
						SeatNumber: item.SeatNumber,
						Status:     status,
					}

					if err := repos.SeatRepository().Create(seat); err != nil {
						return nil, err
					}

					output.Seats = append(output.Seats, *seat)
				}
			} else {
				// Legacy count-based creation: populate seats 1..rowInput.Seats
				for seatNumber := 1; seatNumber <= rowInput.Seats; seatNumber++ {
					gridCol := seatNumber - 1
					var colID *uuid.UUID
					if c, ok := colByOrder[gridCol]; ok {
						colIDCopy := c.ID
						colID = &colIDCopy
					} else if c, ok := colByOrder[seatNumber]; ok {
						colIDCopy := c.ID
						colID = &colIDCopy
					}

					seat := &domain.Seat{
						ID:         uuid.New(),
						RowID:      row.ID,
						ColumnID:   colID,
						GridCol:    gridCol,
						SeatNumber: seatNumber,
						Status:     domain.SeatStatusAvailable,
					}

					if err := repos.SeatRepository().Create(seat); err != nil {
						return nil, err
					}

					output.Seats = append(output.Seats, *seat)
				}
			}
		}
	}

	return output, nil
}

func validateCreateSeatLayoutInput(input CreateSeatLayoutInput) error {
	if input.UserID == uuid.Nil {
		return ErrInvalidSeatLayout
	}
	if input.EventID == uuid.Nil {
		return ErrInvalidSeatLayout
	}
	if err := validateSeatColumns(input.Columns); err != nil {
		return err
	}
	return validateSeatLayoutStructure(input.LayoutName, input.Sections)
}

func validateSeatColumns(columns []SeatColumnInput) error {
	if len(columns) == 0 {
		return nil
	}

	seenLabels := make(map[string]bool)
	seenOrders := make(map[int]bool)

	for _, col := range columns {
		label := strings.TrimSpace(col.Label)
		if label == "" {
			return ErrInvalidSeatColumn
		}
		lower := strings.ToLower(label)
		if seenLabels[lower] {
			return ErrSeatColumnDuplicateLabel
		}
		seenLabels[lower] = true

		if col.OrderIndex < 0 {
			return ErrInvalidSeatColumn
		}
		if seenOrders[col.OrderIndex] {
			return ErrSeatColumnDuplicateOrder
		}
		seenOrders[col.OrderIndex] = true
	}
	return nil
}

func validateSeatLayoutStructure(
	layoutName string,
	sections []SeatSectionInput,
) error {
	if strings.TrimSpace(layoutName) == "" {
		return ErrInvalidSeatLayout
	}

	if len(sections) == 0 {
		return ErrInvalidSeatLayout
	}

	for _, section := range sections {
		if strings.TrimSpace(section.Name) == "" {
			return ErrInvalidSeatSection
		}

		if section.Price.IsNegative() {
			return ErrInvalidSeatSection
		}

		if len(section.Rows) == 0 {
			return ErrInvalidSeatSection
		}

		for _, row := range section.Rows {
			if strings.TrimSpace(row.RowName) == "" {
				return ErrInvalidSeatRow
			}

			if len(row.SeatItems) > 0 {
				seenNumbers := make(map[int]bool)
				seenCols := make(map[int]bool)
				for _, item := range row.SeatItems {
					if item.SeatNumber <= 0 || seenNumbers[item.SeatNumber] {
						return ErrInvalidSeatRow
					}
					seenNumbers[item.SeatNumber] = true
					if item.GridCol < 0 {
						return ErrInvalidSeatRow
					}
					if seenCols[item.GridCol] {
						return ErrInvalidSeatRow
					}
					seenCols[item.GridCol] = true
				}
			} else if row.Seats <= 0 {
				return ErrInvalidSeatRow
			}
		}
	}

	return nil
}

func (u *SeatLayoutUsecase) hasBookings(
	ctx context.Context,
	tx domain.TransactionRepositories,
	layoutID uuid.UUID,
	eventID uuid.UUID,
) (bool, error) {
	if u.bookingGuard != nil {
		return u.bookingGuard.HasBookings(ctx, eventID)
	}

	count, err := tx.SeatRepository().CountBookedByLayoutID(layoutID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (u *SeatLayoutUsecase) UpdateSeatLayout(
	ctx context.Context,
	input UpdateSeatLayoutInput,
) (*UpdateSeatLayoutOutput, error) {
	u.logger.Info(
		"seat_layout_update_started",
		slog.String("user_id", input.UserID.String()),
		slog.String("event_id", input.EventID.String()),
		slog.String("layout_name", input.LayoutName),
	)

	if err := validateUpdateSeatLayoutInput(input); err != nil {
		u.logger.Warn(
			"seat_layout_update_rejected",
			slog.String("user_id", input.UserID.String()),
			slog.String("event_id", input.EventID.String()),
			slog.String("reason", err.Error()),
		)
		return nil, err
	}

	var output UpdateSeatLayoutOutput

	err := u.transactionManager.WithinTransaction(
		func(repos domain.TransactionRepositories) error {
			organizer, err := repos.OrganizerRepository().FindByUserID(input.UserID)
			if err != nil {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("reason", "organizer lookup failed"),
				)
				return ErrUnauthorizedOrganizer
			}

			if organizer.Status != "ACTIVE" {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("reason", "inactive organizer"),
				)
				return ErrUnauthorizedOrganizer
			}

			event, err := repos.EventRepository().FindByIDForUpdate(input.EventID)
			if err != nil {
				return err
			}

			if event.OrganizerID != organizer.ID {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("reason", "ownership verification failed"),
				)
				return ErrUnauthorizedOrganizer
			}

			if event.Status != domain.EventStatusDraft {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("status", event.Status),
					slog.String("reason", "non-draft event"),
				)
				return ErrSeatLayoutDraftOnly
			}

			if event.EventType != domain.EventTypePhysical && event.EventType != domain.EventTypeHybrid {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("event_type", event.EventType),
					slog.String("reason", "invalid event type"),
				)
				return ErrInvalidSeatLayout
			}

			setting, err := repos.EventSettingRepository().FindByEventID(input.EventID)
			if err != nil {
				return err
			}

			if setting.SeatLayoutType != SeatLayoutTypeSeated {
				u.logger.Warn(
					"seat_layout_update_rejected",
					slog.String("user_id", input.UserID.String()),
					slog.String("event_id", input.EventID.String()),
					slog.String("seat_layout_type", setting.SeatLayoutType),
					slog.String("reason", "general ticketing event cannot update seat layout"),
				)
				return ErrSeatLayoutSeatedOnly
			}

			layout, err := repos.SeatLayoutRepository().FindByEventID(input.EventID)
			if err != nil {
				return err
			}

			// Load existing columns
			existingColumns, err := repos.SeatColumnRepository().FindBySeatLayoutID(layout.ID)
			if err != nil {
				return err
			}
			existingColMap := make(map[uuid.UUID]domain.SeatColumn)
			for _, c := range existingColumns {
				existingColMap[c.ID] = c
			}

			existingSections, err := repos.SeatSectionRepository().FindBySeatLayoutID(layout.ID)
			if err != nil {
				return err
			}

			existingRowsMap := make(map[uuid.UUID][]domain.SeatRow)
			existingSeatsMap := make(map[uuid.UUID][]domain.Seat)

			for _, sec := range existingSections {
				rows, err := repos.SeatRowRepository().FindBySectionID(sec.ID)
				if err != nil {
					return err
				}
				existingRowsMap[sec.ID] = rows

				for _, row := range rows {
					seats, err := repos.SeatRepository().FindByRowID(row.ID)
					if err != nil {
						return err
					}
					existingSeatsMap[row.ID] = seats
				}
			}

			hasBookings, err := u.hasBookings(ctx, repos, layout.ID, input.EventID)
			if err != nil {
				return err
			}

			if hasBookings {
				if isStructuralChange(existingSections, existingRowsMap, existingSeatsMap, existingColumns, input) {
					u.logger.Warn(
						"seat_layout_update_rejected",
						slog.String("user_id", input.UserID.String()),
						slog.String("event_id", input.EventID.String()),
						slog.String("reason", "structural changes rejected when bookings exist"),
					)
					return ErrSeatLayoutStructuralChangesNotAllowed
				}
			}

			// Identify incoming IDs and validate ownership
			existingSectionIDMap := make(map[uuid.UUID]domain.SeatSection)
			for _, sec := range existingSections {
				existingSectionIDMap[sec.ID] = sec
			}

			for _, secInput := range input.Sections {
				if secInput.ID != nil {
					if _, exists := existingSectionIDMap[*secInput.ID]; !exists {
						return ErrSeatLayoutDuplicateID
					}
				}

				for _, rowInput := range secInput.Rows {
					if rowInput.ID != nil {
						found := false
						if secInput.ID != nil {
							for _, r := range existingRowsMap[*secInput.ID] {
								if r.ID == *rowInput.ID {
									found = true
									break
								}
							}
						}
						if !found {
							return ErrSeatLayoutDuplicateID
						}
					}

					for _, seatItem := range rowInput.SeatItems {
						if seatItem.ID != nil {
							found := false
							if rowInput.ID != nil {
								for _, s := range existingSeatsMap[*rowInput.ID] {
									if s.ID == *seatItem.ID {
										found = true
										break
									}
								}
							}
							if !found {
								return ErrSeatLayoutDuplicateID
							}
						}
					}
				}
			}

			// BOOKING SAFETY GUARD: Determine all seat IDs marked for deletion
			inputSectionIDs := make(map[uuid.UUID]bool)
			for _, s := range input.Sections {
				if s.ID != nil {
					inputSectionIDs[*s.ID] = true
				}
			}

			inputRowIDs := make(map[uuid.UUID]bool)
			for _, s := range input.Sections {
				for _, r := range s.Rows {
					if r.ID != nil {
						inputRowIDs[*r.ID] = true
					}
				}
			}

			inputColIDs := make(map[uuid.UUID]bool)
			for _, c := range input.Columns {
				if c.ID != nil {
					inputColIDs[*c.ID] = true
				}
			}

			inputSeatIDs := make(map[uuid.UUID]bool)
			for _, s := range input.Sections {
				for _, r := range s.Rows {
					for _, seat := range r.SeatItems {
						if seat.ID != nil {
							inputSeatIDs[*seat.ID] = true
						}
					}
				}
			}

			var candidateSeatsForDeletion []uuid.UUID

			// 1. Seats in removed sections
			for _, sec := range existingSections {
				if !inputSectionIDs[sec.ID] {
					for _, r := range existingRowsMap[sec.ID] {
						for _, s := range existingSeatsMap[r.ID] {
							candidateSeatsForDeletion = append(candidateSeatsForDeletion, s.ID)
						}
					}
				}
			}

			// 2. Seats in removed rows of retained sections
			for _, sec := range existingSections {
				if inputSectionIDs[sec.ID] {
					for _, r := range existingRowsMap[sec.ID] {
						if !inputRowIDs[r.ID] {
							for _, s := range existingSeatsMap[r.ID] {
								candidateSeatsForDeletion = append(candidateSeatsForDeletion, s.ID)
							}
						}
					}
				}
			}

			// 3. Seats in removed columns
			if len(input.Columns) > 0 {
				for _, col := range existingColumns {
					if !inputColIDs[col.ID] {
						colBookedCount, colErr := repos.SeatRepository().CountBookedOrReservedByColumnID(ctx, col.ID)
						if colErr == nil && colBookedCount > 0 {
							u.logger.WarnContext(
								ctx,
								"seat_layout_column_removal_rejected_booked",
								slog.String("event_id", input.EventID.String()),
								slog.String("column_id", col.ID.String()),
							)
							return ErrSeatLayoutBookedSeatCannotBeRemoved
						}

						seatsInCol, findErr := repos.SeatRepository().FindByColumnID(ctx, col.ID)
						if findErr == nil {
							for _, s := range seatsInCol {
								candidateSeatsForDeletion = append(candidateSeatsForDeletion, s.ID)
							}
						}
					}
				}
			}

			// 4. Seats explicitly omitted from SeatItems in retained rows
			for _, secInput := range input.Sections {
				if secInput.ID != nil {
					for _, rowInput := range secInput.Rows {
						if rowInput.ID != nil && len(rowInput.SeatItems) > 0 {
							for _, es := range existingSeatsMap[*rowInput.ID] {
								if !inputSeatIDs[es.ID] {
									candidateSeatsForDeletion = append(candidateSeatsForDeletion, es.ID)
								}
							}
						}
					}
				}
			}

			// Verify candidate deleted seats against bookings and reservations
			if len(candidateSeatsForDeletion) > 0 {
				bookedOrReservedCount, countErr := repos.SeatRepository().CountBookedOrReservedBySeatIDs(ctx, candidateSeatsForDeletion)
				if countErr != nil {
					return countErr
				}
				if bookedOrReservedCount > 0 {
					u.logger.Warn(
						"seat_layout_update_blocked_by_bookings",
						slog.String("event_id", input.EventID.String()),
						slog.Int64("booked_seat_count", bookedOrReservedCount),
					)
					return ErrSeatLayoutBookedSeatCannotBeRemoved
				}
			}

			// Update Layout name
			layout.LayoutName = strings.TrimSpace(input.LayoutName)
			layout.UpdatedAt = time.Now()
			if err := repos.SeatLayoutRepository().Update(layout); err != nil {
				return err
			}
			output.Layout = layout

			// Process Column Updates & Deletions
			finalColumns := make([]domain.SeatColumn, 0)
			colByOrder := make(map[int]domain.SeatColumn)

			if len(input.Columns) > 0 {
				// Delete removed columns
				for _, existingCol := range existingColumns {
					if !inputColIDs[existingCol.ID] {
						// Delete seats remaining in this column (already confirmed unbooked)
						seatsInCol, _ := repos.SeatRepository().FindByColumnID(ctx, existingCol.ID)
						for _, s := range seatsInCol {
							_ = repos.SeatRepository().Delete(s.ID)
						}
						if err := repos.SeatColumnRepository().Delete(existingCol.ID); err != nil {
							return err
						}
					}
				}

				// Create / Update columns
				for idx, colIn := range input.Columns {
					label := strings.TrimSpace(colIn.Label)
					if label == "" {
						label = GenerateDefaultAlphabeticLabel(idx + 1)
					}
					orderIdx := colIn.OrderIndex
					if orderIdx <= 0 {
						orderIdx = idx + 1
					}

					var currentCol *domain.SeatColumn
					if colIn.ID != nil && existingColMap[*colIn.ID].ID != uuid.Nil {
						existingCol := existingColMap[*colIn.ID]
						existingCol.Label = label
						existingCol.OrderIndex = orderIdx
						existingCol.UpdatedAt = time.Now()
						if err := repos.SeatColumnRepository().Update(&existingCol); err != nil {
							return err
						}
						currentCol = &existingCol
					} else {
						newCol := &domain.SeatColumn{
							ID:           uuid.New(),
							SeatLayoutID: layout.ID,
							Label:        label,
							OrderIndex:   orderIdx,
						}
						if err := repos.SeatColumnRepository().Create(newCol); err != nil {
							return err
						}
						currentCol = newCol
					}

					finalColumns = append(finalColumns, *currentCol)
					colByOrder[currentCol.OrderIndex] = *currentCol
				}
			} else {
				// Keep existing columns or auto-generate
				if len(existingColumns) > 0 {
					finalColumns = existingColumns
					for _, c := range existingColumns {
						colByOrder[c.OrderIndex] = c
					}
				}
			}

			output.Columns = finalColumns

			// Delete removed sections
			for _, existingSec := range existingSections {
				if !inputSectionIDs[existingSec.ID] {
					for _, r := range existingRowsMap[existingSec.ID] {
						if err := repos.SeatRepository().DeleteByRowID(r.ID); err != nil {
							return err
						}
						if err := repos.SeatRowRepository().Delete(r.ID); err != nil {
							return err
						}
					}
					if err := repos.SeatSectionRepository().Delete(existingSec.ID); err != nil {
						return err
					}
				}
			}

			// Update or Create Sections
			for _, secInput := range input.Sections {
				var currentSec *domain.SeatSection

				if secInput.ID != nil {
					existingSec := existingSectionIDMap[*secInput.ID]
					existingSec.Name = strings.TrimSpace(secInput.Name)
					existingSec.Price = secInput.Price
					existingSec.UpdatedAt = time.Now()

					if err := repos.SeatSectionRepository().Update(&existingSec); err != nil {
						return err
					}
					currentSec = &existingSec
				} else {
					newSec := &domain.SeatSection{
						ID:           uuid.New(),
						SeatLayoutID: layout.ID,
						Name:         strings.TrimSpace(secInput.Name),
						Price:        secInput.Price,
					}
					if err := repos.SeatSectionRepository().Create(newSec); err != nil {
						return err
					}
					currentSec = newSec
				}

				output.Sections = append(output.Sections, *currentSec)

				// Delete removed rows in this section
				if secInput.ID != nil {
					for _, existingRow := range existingRowsMap[*secInput.ID] {
						if !inputRowIDs[existingRow.ID] {
							if err := repos.SeatRepository().DeleteByRowID(existingRow.ID); err != nil {
								return err
							}
							if err := repos.SeatRowRepository().Delete(existingRow.ID); err != nil {
								return err
							}
						}
					}
				}

				// Update or Create Rows
				for _, rowInput := range secInput.Rows {
					var currentRow *domain.SeatRow

					if rowInput.ID != nil {
						var existingRow *domain.SeatRow
						if secInput.ID != nil {
							for _, r := range existingRowsMap[*secInput.ID] {
								if r.ID == *rowInput.ID {
									rowCopy := r
									existingRow = &rowCopy
									break
								}
							}
						}
						if existingRow != nil {
							existingRow.RowName = strings.TrimSpace(rowInput.RowName)
							if err := repos.SeatRowRepository().Update(existingRow); err != nil {
								return err
							}
							currentRow = existingRow
						}
					}

					if currentRow == nil {
						newRow := &domain.SeatRow{
							ID:        uuid.New(),
							SectionID: currentSec.ID,
							RowName:   strings.TrimSpace(rowInput.RowName),
						}
						if err := repos.SeatRowRepository().Create(newRow); err != nil {
							return err
						}
						currentRow = newRow
					}

					output.Rows = append(output.Rows, *currentRow)

					existingSeats := existingSeatsMap[currentRow.ID]

					if len(rowInput.SeatItems) > 0 {
						// Delete omitted seats in this row (creating gaps)
						rowInputSeatIDs := make(map[uuid.UUID]bool)
						for _, s := range rowInput.SeatItems {
							if s.ID != nil {
								rowInputSeatIDs[*s.ID] = true
							}
						}

						for _, es := range existingSeats {
							if !rowInputSeatIDs[es.ID] {
								if err := repos.SeatRepository().Delete(es.ID); err != nil {
									return err
								}
							}
						}

						// Update or insert seat items
						for _, seatItem := range rowInput.SeatItems {
							status := domain.SeatStatusAvailable
							if seatItem.Status != "" {
								status = strings.ToUpper(seatItem.Status)
							}

							gridCol := seatItem.GridCol

							var colID *uuid.UUID
							if seatItem.ColumnID != nil {
								colID = seatItem.ColumnID
							} else if c, ok := colByOrder[gridCol]; ok {
								colIDCopy := c.ID
								colID = &colIDCopy
							}

							if seatItem.ID != nil {
								var existingSeat *domain.Seat
								for _, es := range existingSeats {
									if es.ID == *seatItem.ID {
										seatCopy := es
										existingSeat = &seatCopy
										break
									}
								}
								if existingSeat != nil {
									existingSeat.SeatNumber = seatItem.SeatNumber
									existingSeat.GridCol = gridCol
									existingSeat.ColumnID = colID
									existingSeat.Status = status
									if err := repos.SeatRepository().Update(existingSeat); err != nil {
										return err
									}
									output.Seats = append(output.Seats, *existingSeat)
								}
							} else {
								newSeat := &domain.Seat{
									ID:         uuid.New(),
									RowID:      currentRow.ID,
									ColumnID:   colID,
									GridCol:    gridCol,
									SeatNumber: seatItem.SeatNumber,
									Status:     status,
								}
								if err := repos.SeatRepository().Create(newSeat); err != nil {
									return err
								}
								output.Seats = append(output.Seats, *newSeat)
							}
						}
					} else {
						// Fallback legacy count-based logic
						desiredCount := rowInput.Seats
						existingCount := len(existingSeats)

						if existingCount == desiredCount {
							for _, es := range existingSeats {
								output.Seats = append(output.Seats, es)
							}
						} else if existingCount > desiredCount {
							for i, es := range existingSeats {
								if i < desiredCount {
									output.Seats = append(output.Seats, es)
								} else {
									if err := repos.SeatRepository().Delete(es.ID); err != nil {
										return err
									}
								}
							}
						} else {
							for _, es := range existingSeats {
								output.Seats = append(output.Seats, es)
							}
							for seatNum := existingCount + 1; seatNum <= desiredCount; seatNum++ {
								var colID *uuid.UUID
								if c, ok := colByOrder[seatNum]; ok {
									colIDCopy := c.ID
									colID = &colIDCopy
								}
								newSeat := &domain.Seat{
									ID:         uuid.New(),
									RowID:      currentRow.ID,
									ColumnID:   colID,
									GridCol:    seatNum,
									SeatNumber: seatNum,
									Status:     domain.SeatStatusAvailable,
								}
								if err := repos.SeatRepository().Create(newSeat); err != nil {
									return err
								}
								output.Seats = append(output.Seats, *newSeat)
							}
						}
					}
				}
			}

			return nil
		},
	)

	if err != nil {
		u.logger.Error(
			"seat_layout_update_failed",
			slog.String("user_id", input.UserID.String()),
			slog.String("event_id", input.EventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.Info(
		"seat_layout_updated_successfully",
		slog.String("user_id", input.UserID.String()),
		slog.String("event_id", input.EventID.String()),
		slog.String("seat_layout_id", output.Layout.ID.String()),
		slog.Int("column_count", len(output.Columns)),
		slog.Int("section_count", len(output.Sections)),
		slog.Int("row_count", len(output.Rows)),
		slog.Int("seat_count", len(output.Seats)),
	)

	return &output, nil
}

func validateUpdateSeatLayoutInput(input UpdateSeatLayoutInput) error {
	if input.UserID == uuid.Nil {
		return ErrInvalidSeatLayout
	}
	if input.EventID == uuid.Nil {
		return ErrInvalidSeatLayout
	}
	if strings.TrimSpace(input.LayoutName) == "" {
		return ErrInvalidSeatLayout
	}
	if len(input.Sections) == 0 {
		return ErrInvalidSeatLayout
	}

	if err := validateUpdateSeatColumns(input.Columns); err != nil {
		return err
	}

	seenSectionIDs := make(map[uuid.UUID]bool)
	seenSectionNames := make(map[string]bool)
	seenRowIDs := make(map[uuid.UUID]bool)
	seenSeatIDs := make(map[uuid.UUID]bool)

	for _, section := range input.Sections {
		name := strings.TrimSpace(section.Name)
		if name == "" {
			return ErrInvalidSeatSection
		}
		lowerName := strings.ToLower(name)
		if seenSectionNames[lowerName] {
			return ErrInvalidSeatSection
		}
		seenSectionNames[lowerName] = true

		if section.Price.IsNegative() {
			return ErrInvalidSeatSection
		}
		if section.ID != nil {
			if *section.ID == uuid.Nil || seenSectionIDs[*section.ID] {
				return ErrSeatLayoutDuplicateID
			}
			seenSectionIDs[*section.ID] = true
		}
		if len(section.Rows) == 0 {
			return ErrInvalidSeatSection
		}

		seenRowNames := make(map[string]bool)
		for _, row := range section.Rows {
			rowName := strings.TrimSpace(row.RowName)
			if rowName == "" {
				return ErrInvalidSeatRow
			}
			lowerRowName := strings.ToLower(rowName)
			if seenRowNames[lowerRowName] {
				return ErrInvalidSeatRow
			}
			seenRowNames[lowerRowName] = true

			if row.ID != nil {
				if *row.ID == uuid.Nil || seenRowIDs[*row.ID] {
					return ErrSeatLayoutDuplicateID
				}
				seenRowIDs[*row.ID] = true
			}

			if len(row.SeatItems) > 0 {
				seenSeatNumbers := make(map[int]bool)
				seenGridCols := make(map[int]bool)
				for _, seat := range row.SeatItems {
					if seat.SeatNumber <= 0 || seenSeatNumbers[seat.SeatNumber] {
						return ErrInvalidSeatRow
					}
					seenSeatNumbers[seat.SeatNumber] = true

					if seat.GridCol < 0 {
						return ErrInvalidSeatRow
					}
					if seenGridCols[seat.GridCol] {
						return ErrInvalidSeatRow
					}
					seenGridCols[seat.GridCol] = true

					if seat.ID != nil {
						if *seat.ID == uuid.Nil || seenSeatIDs[*seat.ID] {
							return ErrSeatLayoutDuplicateID
						}
						seenSeatIDs[*seat.ID] = true
					}

					if seat.Status != "" {
						statusUpper := strings.ToUpper(seat.Status)
						if statusUpper != domain.SeatStatusAvailable &&
							statusUpper != domain.SeatStatusReserved &&
							statusUpper != domain.SeatStatusBooked &&
							statusUpper != domain.SeatStatusDisabled {
							return ErrSeatLayoutInvalidStatus
						}
					}
				}
			} else {
				if row.Seats <= 0 {
					return ErrInvalidSeatRow
				}
			}
		}
	}
	return nil
}

func validateUpdateSeatColumns(columns []UpdateSeatColumnInput) error {
	if len(columns) == 0 {
		return nil
	}

	seenLabels := make(map[string]bool)
	seenOrders := make(map[int]bool)
	seenColIDs := make(map[uuid.UUID]bool)

	for _, col := range columns {
		label := strings.TrimSpace(col.Label)
		if label == "" {
			return ErrInvalidSeatColumn
		}
		lower := strings.ToLower(label)
		if seenLabels[lower] {
			return ErrSeatColumnDuplicateLabel
		}
		seenLabels[lower] = true

		if col.OrderIndex < 0 {
			return ErrInvalidSeatColumn
		}
		if seenOrders[col.OrderIndex] {
			return ErrSeatColumnDuplicateOrder
		}
		seenOrders[col.OrderIndex] = true

		if col.ID != nil {
			if *col.ID == uuid.Nil || seenColIDs[*col.ID] {
				return ErrSeatLayoutDuplicateID
			}
			seenColIDs[*col.ID] = true
		}
	}

	return nil
}

func isStructuralChange(
	existingSections []domain.SeatSection,
	existingRowsMap map[uuid.UUID][]domain.SeatRow,
	existingSeatsMap map[uuid.UUID][]domain.Seat,
	existingColumns []domain.SeatColumn,
	input UpdateSeatLayoutInput,
) bool {
	if len(input.Columns) != len(existingColumns) {
		return true
	}

	if len(input.Columns) > 0 && len(existingColumns) > 0 {
		existingColMap := make(map[uuid.UUID]domain.SeatColumn)
		for _, col := range existingColumns {
			existingColMap[col.ID] = col
		}

		for _, colInput := range input.Columns {
			if colInput.ID == nil {
				return true
			}
			existingCol, ok := existingColMap[*colInput.ID]
			if !ok {
				return true
			}
			if colInput.OrderIndex != existingCol.OrderIndex {
				return true
			}
		}
	}

	if len(existingSections) != len(input.Sections) {
		return true
	}

	existingSecMap := make(map[uuid.UUID]domain.SeatSection)
	for _, sec := range existingSections {
		existingSecMap[sec.ID] = sec
	}

	for _, secInput := range input.Sections {
		var existingSec *domain.SeatSection
		if secInput.ID != nil {
			if s, ok := existingSecMap[*secInput.ID]; ok {
				existingSec = &s
			}
		} else {
			for _, s := range existingSections {
				if strings.EqualFold(s.Name, strings.TrimSpace(secInput.Name)) {
					secCopy := s
					existingSec = &secCopy
					break
				}
			}
		}
		if existingSec == nil {
			return true
		}

		if !strings.EqualFold(existingSec.Name, strings.TrimSpace(secInput.Name)) {
			return true
		}

		existingRows := existingRowsMap[existingSec.ID]
		if len(existingRows) != len(secInput.Rows) {
			return true
		}

		existingRowMap := make(map[uuid.UUID]domain.SeatRow)
		for _, r := range existingRows {
			existingRowMap[r.ID] = r
		}

		for _, rowInput := range secInput.Rows {
			var existingRow *domain.SeatRow
			if rowInput.ID != nil {
				if r, ok := existingRowMap[*rowInput.ID]; ok {
					existingRow = &r
				}
			} else {
				for _, r := range existingRows {
					if strings.EqualFold(r.RowName, strings.TrimSpace(rowInput.RowName)) {
						rowCopy := r
						existingRow = &rowCopy
						break
					}
				}
			}
			if existingRow == nil {
				return true
			}

			if !strings.EqualFold(existingRow.RowName, strings.TrimSpace(rowInput.RowName)) {
				return true
			}

			existingSeats := existingSeatsMap[existingRow.ID]
			expectedSeatCount := rowInput.Seats
			if len(rowInput.SeatItems) > 0 {
				expectedSeatCount = len(rowInput.SeatItems)
			}
			if len(existingSeats) != expectedSeatCount {
				return true
			}

			if len(rowInput.SeatItems) > 0 {
				existingSeatByID := make(map[uuid.UUID]domain.Seat)
				existingSeatByNumber := make(map[int]domain.Seat)
				for _, s := range existingSeats {
					existingSeatByID[s.ID] = s
					existingSeatByNumber[s.SeatNumber] = s
				}

				for _, seatInput := range rowInput.SeatItems {
					var existingSeat *domain.Seat
					if seatInput.ID != nil {
						if s, ok := existingSeatByID[*seatInput.ID]; ok {
							existingSeat = &s
						}
					} else {
						if s, ok := existingSeatByNumber[seatInput.SeatNumber]; ok {
							existingSeat = &s
						}
					}

					if existingSeat == nil {
						return true
					}

					if seatInput.GridCol != existingSeat.GridCol {
						return true
					}

					if seatInput.ColumnID != nil && existingSeat.ColumnID != nil {
						if *seatInput.ColumnID != *existingSeat.ColumnID {
							return true
						}
					} else if (seatInput.ColumnID == nil) != (existingSeat.ColumnID == nil) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (u *SeatLayoutUsecase) GetPublicSeatLayout(
	ctx context.Context,
	eventID uuid.UUID,
) (*domain.PublicSeatLayout, error) {
	if eventID == uuid.Nil {
		return nil, ErrSeatLayoutEventNotFound
	}

	var output domain.PublicSeatLayout

	err := u.transactionManager.WithinTransaction(
		func(repos domain.TransactionRepositories) error {
			event, err := repos.EventRepository().FindByID(eventID)
			if err != nil {
				return err
			}

			if event.Status != domain.EventStatusPublished || event.Visibility != "PUBLIC" {
				return domain.ErrEventNotPublic
			}

			if event.EventType == domain.EventTypeOnline {
				return ErrSeatLayoutSeatedOnly
			}

			setting, err := repos.EventSettingRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			if setting.SeatLayoutType != SeatLayoutTypeSeated {
				return ErrSeatLayoutSeatedOnly
			}

			layout, err := repos.SeatLayoutRepository().FindByEventID(eventID)
			if err != nil {
				return err
			}

			// Load layout columns
			columns, err := repos.SeatColumnRepository().FindBySeatLayoutID(layout.ID)
			if err != nil {
				return err
			}

			publicColumns := make([]domain.PublicSeatColumn, 0, len(columns))
			colLabelMap := make(map[uuid.UUID]string)
			for _, c := range columns {
				publicColumns = append(publicColumns, domain.PublicSeatColumn{
					ID:         c.ID,
					Label:      c.Label,
					OrderIndex: c.OrderIndex,
				})
				colLabelMap[c.ID] = c.Label
			}

			sections, err := repos.SeatSectionRepository().FindBySeatLayoutID(layout.ID)
			if err != nil {
				return err
			}

			publicSections := make([]domain.PublicSeatSection, 0, len(sections))

			for _, sec := range sections {
				rows, err := repos.SeatRowRepository().FindBySectionID(sec.ID)
				if err != nil {
					return err
				}

				publicRows := make([]domain.PublicSeatRow, 0, len(rows))

				for _, r := range rows {
					seats, err := repos.SeatRepository().FindByRowID(r.ID)
					if err != nil {
						return err
					}

					publicSeats := make([]domain.PublicSeat, 0, len(seats))
					for _, s := range seats {
						colLabel := ""
						if s.ColumnID != nil {
							colLabel = colLabelMap[*s.ColumnID]
						}
						if colLabel == "" {
							colLabel = GenerateDefaultAlphabeticLabel(s.GridCol)
						}

						publicSeats = append(publicSeats, domain.PublicSeat{
							ID:          s.ID,
							ColumnID:    s.ColumnID,
							GridCol:     s.GridCol,
							ColumnLabel: colLabel,
							SeatNumber:  s.SeatNumber,
							Status:      s.Status,
						})
					}

					publicRows = append(publicRows, domain.PublicSeatRow{
						ID:      r.ID,
						RowName: r.RowName,
						Seats:   publicSeats,
					})
				}

				publicSections = append(publicSections, domain.PublicSeatSection{
					ID:    sec.ID,
					Name:  sec.Name,
					Price: sec.Price.InexactFloat64(),
					Rows:  publicRows,
				})
			}

			// If columns were empty in DB (legacy layout), auto-generate default column headers
			if len(publicColumns) == 0 {
				maxCol := 0
				for _, sec := range publicSections {
					for _, r := range sec.Rows {
						for _, s := range r.Seats {
							if s.GridCol > maxCol {
								maxCol = s.GridCol
							}
						}
					}
				}
				for c := 1; c <= maxCol; c++ {
					publicColumns = append(publicColumns, domain.PublicSeatColumn{
						ID:         uuid.Nil,
						Label:      GenerateDefaultAlphabeticLabel(c),
						OrderIndex: c,
					})
				}
			}

			output = domain.PublicSeatLayout{
				ID:         layout.ID,
				EventID:    layout.EventID,
				LayoutName: layout.LayoutName,
				Columns:    publicColumns,
				Sections:   publicSections,
			}

			return nil
		},
	)

	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get public seat layout",
			slog.String("event_id", eventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return &output, nil
}

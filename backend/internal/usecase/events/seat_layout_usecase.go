package events

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrSeatLayoutEventNotFound               = errors.New("event not found")
	ErrInvalidSeatLayout                     = errors.New("invalid seat layout")
	ErrInvalidSeatSection                    = errors.New("invalid seat section")
	ErrInvalidSeatRow                        = errors.New("invalid seat row")
	ErrSeatLayoutDraftOnly                   = errors.New("seat layout can only be modified for draft events")
	ErrSeatLayoutSeatedOnly                  = errors.New("seat layout is only supported for seated events")
	ErrSeatLayoutStructuralChangesNotAllowed = errors.New("structural changes to seat layout are not allowed once bookings exist")
	ErrSeatLayoutDuplicateID                 = errors.New("duplicate or conflicting seat layout ID")
	ErrSeatLayoutInvalidStatus               = errors.New("invalid seat status")
)

type BookingGuard interface {
	HasBookings(ctx context.Context, eventID uuid.UUID) (bool, error)
}

type CreateSeatLayoutInput struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	Sections   []SeatSectionInput
}

type SeatSectionInput struct {
	Name  string
	Price float64
	Rows  []SeatRowInput
}

type SeatRowInput struct {
	RowName string
	Seats   int
}

type CreateSeatLayoutOutput struct {
	Layout   *domain.SeatLayout
	Sections []domain.SeatSection
	Rows     []domain.SeatRow
	Seats    []domain.Seat
}

type UpdateSeatLayoutInput struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	LayoutName string
	Sections   []UpdateSeatSectionInput
}

type UpdateSeatSectionInput struct {
	ID    *uuid.UUID
	Name  string
	Price float64
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
	SeatNumber int
	Status     string
}

type UpdateSeatLayoutOutput struct {
	Layout   *domain.SeatLayout
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
			organizer, err := repos.OrganizerRepository().
				FindByUserID(input.UserID)
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
			event, err := repos.EventRepository().
				FindByID(input.EventID)
			if err != nil {
				return err
			}

			// events.organizer_id references organizers.id,
			// while input.UserID references users.id.
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
			setting, err := repos.EventSettingRepository().
				FindByEventID(input.EventID)
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
			if _, err := repos.SeatLayoutRepository().
				FindByEventID(input.EventID); err == nil {

				u.logger.Warn(
					"seat_layout_already_exists",
					"event_id", input.EventID,
					"user_id", input.UserID,
				)

				return ErrInvalidSeatLayout

			} else if !errors.Is(err, domain.ErrSeatLayoutNotFound) {
				return err
			}

			// Create seat layout.
			layout := &domain.SeatLayout{
				ID:         uuid.New(),
				EventID:    input.EventID,
				LayoutName: strings.TrimSpace(input.LayoutName),
			}

			if err := repos.SeatLayoutRepository().Create(layout); err != nil {
				return err
			}

			output.Layout = layout

			// Create sections.
			for _, sectionInput := range input.Sections {
				section := &domain.SeatSection{
					ID:           uuid.New(),
					SeatLayoutID: layout.ID,
					Name:         strings.TrimSpace(sectionInput.Name),
					Price:        sectionInput.Price,
				}

				if err := repos.SeatSectionRepository().Create(section); err != nil {
					return err
				}

				output.Sections = append(
					output.Sections,
					*section,
				)

				// Create rows.
				for _, rowInput := range sectionInput.Rows {
					row := &domain.SeatRow{
						ID:        uuid.New(),
						SectionID: section.ID,
						RowName:   strings.TrimSpace(rowInput.RowName),
					}

					if err := repos.SeatRowRepository().Create(row); err != nil {
						return err
					}

					output.Rows = append(
						output.Rows,
						*row,
					)

					// Create individual seats.
					for seatNumber := 1; seatNumber <= rowInput.Seats; seatNumber++ {
						seat := &domain.Seat{
							ID:         uuid.New(),
							RowID:      row.ID,
							SeatNumber: seatNumber,
							Status:     domain.SeatStatusAvailable,
						}

						if err := repos.SeatRepository().Create(seat); err != nil {
							return err
						}

						output.Seats = append(
							output.Seats,
							*seat,
						)
					}
				}
			}

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

		output.Sections = append(
			output.Sections,
			*section,
		)

		for _, rowInput := range sectionInput.Rows {
			row := &domain.SeatRow{
				ID:        uuid.New(),
				SectionID: section.ID,
				RowName:   strings.TrimSpace(rowInput.RowName),
			}

			if err := repos.SeatRowRepository().Create(row); err != nil {
				return nil, err
			}

			output.Rows = append(
				output.Rows,
				*row,
			)

			for seatNumber := 1; seatNumber <= rowInput.Seats; seatNumber++ {
				seat := &domain.Seat{
					ID:         uuid.New(),
					RowID:      row.ID,
					SeatNumber: seatNumber,
					Status:     domain.SeatStatusAvailable,
				}

				if err := repos.SeatRepository().Create(seat); err != nil {
					return nil, err
				}

				output.Seats = append(
					output.Seats,
					*seat,
				)
			}
		}
	}

	return output, nil
}

func validateCreateSeatLayoutInput(
	input CreateSeatLayoutInput,
) error {
	if input.UserID == uuid.Nil {
		return ErrInvalidSeatLayout
	}

	if input.EventID == uuid.Nil {
		return ErrInvalidSeatLayout
	}

	return validateSeatLayoutStructure(input.LayoutName, input.Sections)
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

		if section.Price < 0 {
			return ErrInvalidSeatSection
		}

		if len(section.Rows) == 0 {
			return ErrInvalidSeatSection
		}

		for _, row := range section.Rows {
			if strings.TrimSpace(row.RowName) == "" {
				return ErrInvalidSeatRow
			}

			if row.Seats <= 0 {
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

			event, err := repos.EventRepository().FindByID(input.EventID)
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
				if isStructuralChange(existingSections, existingRowsMap, existingSeatsMap, input) {
					u.logger.Warn(
						"seat_layout_update_rejected",
						slog.String("user_id", input.UserID.String()),
						slog.String("event_id", input.EventID.String()),
						slog.String("reason", "structural changes rejected when bookings exist"),
					)
					return ErrSeatLayoutStructuralChangesNotAllowed
				}
			}

			existingSectionIDMap := make(map[uuid.UUID]domain.SeatSection)
			for _, sec := range existingSections {
				existingSectionIDMap[sec.ID] = sec
			}

			for _, secInput := range input.Sections {
				if secInput.ID != nil {
					if _, exists := existingSectionIDMap[*secInput.ID]; !exists {
						u.logger.Warn("seat_layout_update_rejected", slog.String("reason", "section ID does not belong to layout"))
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
							u.logger.Warn("seat_layout_update_rejected", slog.String("reason", "row ID does not belong to section"))
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
								u.logger.Warn("seat_layout_update_rejected", slog.String("reason", "seat ID does not belong to row"))
								return ErrSeatLayoutDuplicateID
							}
						}
					}
				}
			}

			layout.LayoutName = strings.TrimSpace(input.LayoutName)
			layout.UpdatedAt = time.Now()
			if err := repos.SeatLayoutRepository().Update(layout); err != nil {
				return err
			}
			output.Layout = layout

			inputSectionIDs := make(map[uuid.UUID]bool)
			for _, s := range input.Sections {
				if s.ID != nil {
					inputSectionIDs[*s.ID] = true
				}
			}

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

				inputRowIDs := make(map[uuid.UUID]bool)
				for _, r := range secInput.Rows {
					if r.ID != nil {
						inputRowIDs[*r.ID] = true
					}
				}

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
						existingRow.RowName = strings.TrimSpace(rowInput.RowName)
						if err := repos.SeatRowRepository().Update(existingRow); err != nil {
							return err
						}
						currentRow = existingRow
					} else {
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
						inputSeatIDs := make(map[uuid.UUID]bool)
						for _, s := range rowInput.SeatItems {
							if s.ID != nil {
								inputSeatIDs[*s.ID] = true
							}
						}

						for _, es := range existingSeats {
							if !inputSeatIDs[es.ID] {
								if err := repos.SeatRepository().Delete(es.ID); err != nil {
									return err
								}
							}
						}

						for _, seatItem := range rowInput.SeatItems {
							status := domain.SeatStatusAvailable
							if seatItem.Status != "" {
								status = strings.ToUpper(seatItem.Status)
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
								existingSeat.SeatNumber = seatItem.SeatNumber
								existingSeat.Status = status
								if err := repos.SeatRepository().Update(existingSeat); err != nil {
									return err
								}
								output.Seats = append(output.Seats, *existingSeat)
							} else {
								newSeat := &domain.Seat{
									ID:         uuid.New(),
									RowID:      currentRow.ID,
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
								newSeat := &domain.Seat{
									ID:         uuid.New(),
									RowID:      currentRow.ID,
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

		if section.Price < 0 {
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
				for _, seat := range row.SeatItems {
					if seat.SeatNumber <= 0 || seenSeatNumbers[seat.SeatNumber] {
						return ErrInvalidSeatRow
					}
					seenSeatNumbers[seat.SeatNumber] = true

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

func isStructuralChange(
	existingSections []domain.SeatSection,
	existingRowsMap map[uuid.UUID][]domain.SeatRow,
	existingSeatsMap map[uuid.UUID][]domain.Seat,
	input UpdateSeatLayoutInput,
) bool {
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
		}
	}

	return false
}

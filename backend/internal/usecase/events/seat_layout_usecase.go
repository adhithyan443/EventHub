package events

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrSeatLayoutEventNotFound = errors.New("event not found")
	ErrInvalidSeatLayout       = errors.New("invalid seat layout")
	ErrInvalidSeatSection      = errors.New("invalid seat section")
	ErrInvalidSeatRow          = errors.New("invalid seat row")
)

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

type SeatLayoutUsecase struct {
	transactionManager domain.TransactionManager
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

func validateCreateSeatLayoutInput(
	input CreateSeatLayoutInput,
) error {
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

	for _, section := range input.Sections {
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

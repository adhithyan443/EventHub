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

	err := u.transactionManager.WithinTransaction(func(repos domain.TransactionRepositories) error {
		event, err := repos.EventRepository().FindByID(input.EventID)
		if err != nil {
			return err
		}

		if event.OrganizerID != input.UserID {
			u.logger.Warn(
				"seat_layout_unauthorized",
				"event_id", input.EventID,
				"user_id", input.UserID,
			)

			return ErrUnauthorizedOrganizer
		}

		if event.EventType != domain.EventTypePhysical &&
			event.EventType != domain.EventTypeHybrid {
			return ErrInvalidSeatLayout
		}

		setting, err := repos.EventSettingRepository().FindByEventID(input.EventID)
		if err != nil {
			return err
		}

		if setting.SeatLayoutType != SeatLayoutTypeSeated {
			return ErrInvalidSeatLayout
		}

		if _, err := repos.SeatLayoutRepository().FindByEventID(input.EventID); err == nil {
			return ErrInvalidSeatLayout
		} else if !errors.Is(err, domain.ErrSeatLayoutNotFound) {
			return err
		}

		layout := &domain.SeatLayout{
			ID:         uuid.New(),
			EventID:    input.EventID,
			LayoutName: strings.TrimSpace(input.LayoutName),
		}

		if err := repos.SeatLayoutRepository().Create(layout); err != nil {
			return err
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
				return err
			}

			output.Sections = append(output.Sections, *section)

			for _, rowInput := range sectionInput.Rows {
				row := &domain.SeatRow{
					ID:        uuid.New(),
					SectionID: section.ID,
					RowName:   strings.TrimSpace(rowInput.RowName),
				}

				if err := repos.SeatRowRepository().Create(row); err != nil {
					return err
				}

				output.Rows = append(output.Rows, *row)

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

					output.Seats = append(output.Seats, *seat)
				}
			}
		}

		return nil
	})

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

func validateCreateSeatLayoutInput(input CreateSeatLayoutInput) error {
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

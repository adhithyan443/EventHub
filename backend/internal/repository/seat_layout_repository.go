package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeatLayoutRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatLayoutRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *SeatLayoutRepository {
	return &SeatLayoutRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SeatLayoutRepository) Create(layout *domain.SeatLayout) error {
	layoutModel := toSeatLayoutModel(layout)

	if err := r.db.Create(layoutModel).Error; err != nil {
		r.logger.Error(
			"seat_layout_create_failed",
			"event_id", layout.EventID,
			"error", err,
		)

		return err
	}

	*layout = *toSeatLayoutDomain(layoutModel)

	r.logger.Info(
		"seat_layout_created",
		"seat_layout_id", layout.ID,
		"event_id", layout.EventID,
	)

	return nil
}

func (r *SeatLayoutRepository) FindByEventID(
	eventID uuid.UUID,
) (*domain.SeatLayout, error) {
	var layoutModel models.SeatLayoutModel

	if err := r.db.
		Where("event_id = ?", eventID).
		First(&layoutModel).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSeatLayoutNotFound
		}

		r.logger.Error(
			"seat_layout_find_by_event_id_failed",
			"event_id", eventID,
			"error", err,
		)

		return nil, err
	}

	return toSeatLayoutDomain(&layoutModel), nil
}

func toSeatLayoutModel(layout *domain.SeatLayout) *models.SeatLayoutModel {
	return &models.SeatLayoutModel{
		ID:         layout.ID,
		EventID:    layout.EventID,
		LayoutName: layout.LayoutName,
		CreatedAt:  layout.CreatedAt,
		UpdatedAt:  layout.UpdatedAt,
	}
}

func toSeatLayoutDomain(
	layoutModel *models.SeatLayoutModel,
) *domain.SeatLayout {
	return &domain.SeatLayout{
		ID:         layoutModel.ID,
		EventID:    layoutModel.EventID,
		LayoutName: layoutModel.LayoutName,
		CreatedAt:  layoutModel.CreatedAt,
		UpdatedAt:  layoutModel.UpdatedAt,
	}
}

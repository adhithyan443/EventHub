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

func (r *SeatLayoutRepository) DeleteByEventID(eventID uuid.UUID) error {
	var layout models.SeatLayoutModel
	if err := r.db.Where("event_id = ?", eventID).First(&layout).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		r.logger.Error(
			"seat_layout_find_for_delete_failed",
			"event_id", eventID,
			"error", err,
		)
		return err
	}

	var sectionIDs []uuid.UUID
	if err := r.db.Model(&models.SeatSectionModel{}).
		Where("seat_layout_id = ?", layout.ID).
		Pluck("id", &sectionIDs).Error; err != nil {
		r.logger.Error(
			"seat_sections_pluck_failed",
			"seat_layout_id", layout.ID,
			"error", err,
		)
		return err
	}

	if len(sectionIDs) > 0 {
		var rowIDs []uuid.UUID
		if err := r.db.Model(&models.SeatRowModel{}).
			Where("section_id IN ?", sectionIDs).
			Pluck("id", &rowIDs).Error; err != nil {
			r.logger.Error(
				"seat_rows_pluck_failed",
				"error", err,
			)
			return err
		}

		if len(rowIDs) > 0 {
			if err := r.db.Where("row_id IN ?", rowIDs).Delete(&models.SeatModel{}).Error; err != nil {
				r.logger.Error(
					"seats_delete_failed",
					"error", err,
				)
				return err
			}

			if err := r.db.Where("id IN ?", rowIDs).Delete(&models.SeatRowModel{}).Error; err != nil {
				r.logger.Error(
					"seat_rows_delete_failed",
					"error", err,
				)
				return err
			}
		}

		if err := r.db.Where("id IN ?", sectionIDs).Delete(&models.SeatSectionModel{}).Error; err != nil {
			r.logger.Error(
				"seat_sections_delete_failed",
				"error", err,
			)
			return err
		}
	}

	if err := r.db.Where("id = ?", layout.ID).Delete(&models.SeatLayoutModel{}).Error; err != nil {
		r.logger.Error(
			"seat_layout_delete_failed",
			"seat_layout_id", layout.ID,
			"error", err,
		)
		return err
	}

	r.logger.Info(
		"seat_layout_deleted",
		"seat_layout_id", layout.ID,
		"event_id", eventID,
	)

	return nil
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

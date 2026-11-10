package repository

import (
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeatColumnRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatColumnRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *SeatColumnRepository {
	return &SeatColumnRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SeatColumnRepository) Create(column *domain.SeatColumn) error {
	colModel := toSeatColumnModel(column)

	if err := r.db.Create(colModel).Error; err != nil {
		r.logger.Error(
			"seat_column_create_failed",
			"seat_layout_id", column.SeatLayoutID,
			"label", column.Label,
			"error", err,
		)
		return err
	}

	*column = *toSeatColumnDomain(colModel)

	r.logger.Info(
		"seat_column_created",
		"column_id", column.ID,
		"seat_layout_id", column.SeatLayoutID,
		"label", column.Label,
		"order_index", column.OrderIndex,
	)

	return nil
}

func (r *SeatColumnRepository) CreateBatch(columns []domain.SeatColumn) error {
	if len(columns) == 0 {
		return nil
	}

	colModels := make([]models.SeatColumnModel, 0, len(columns))
	for i := range columns {
		colModels = append(colModels, *toSeatColumnModel(&columns[i]))
	}

	if err := r.db.Create(&colModels).Error; err != nil {
		r.logger.Error(
			"seat_column_create_batch_failed",
			"count", len(columns),
			"error", err,
		)
		return err
	}

	for i := range colModels {
		columns[i] = *toSeatColumnDomain(&colModels[i])
	}

	return nil
}

func (r *SeatColumnRepository) Update(column *domain.SeatColumn) error {
	colModel := toSeatColumnModel(column)

	if err := r.db.Save(colModel).Error; err != nil {
		r.logger.Error(
			"seat_column_update_failed",
			"column_id", column.ID,
			"seat_layout_id", column.SeatLayoutID,
			"error", err,
		)
		return err
	}

	*column = *toSeatColumnDomain(colModel)

	r.logger.Info(
		"seat_column_updated",
		"column_id", column.ID,
		"seat_layout_id", column.SeatLayoutID,
		"label", column.Label,
		"order_index", column.OrderIndex,
	)

	return nil
}

func (r *SeatColumnRepository) Delete(id uuid.UUID) error {
	if err := r.db.Where("id = ?", id).Delete(&models.SeatColumnModel{}).Error; err != nil {
		r.logger.Error(
			"seat_column_delete_failed",
			"column_id", id,
			"error", err,
		)
		return err
	}

	r.logger.Info(
		"seat_column_deleted",
		"column_id", id,
	)

	return nil
}

func (r *SeatColumnRepository) DeleteBySeatLayoutID(layoutID uuid.UUID) error {
	if err := r.db.Where("seat_layout_id = ?", layoutID).Delete(&models.SeatColumnModel{}).Error; err != nil {
		r.logger.Error(
			"seat_columns_delete_by_layout_failed",
			"seat_layout_id", layoutID,
			"error", err,
		)
		return err
	}

	r.logger.Info(
		"seat_columns_deleted_by_layout",
		"seat_layout_id", layoutID,
	)

	return nil
}

func (r *SeatColumnRepository) FindBySeatLayoutID(
	layoutID uuid.UUID,
) ([]domain.SeatColumn, error) {
	var colModels []models.SeatColumnModel

	if err := r.db.
		Where("seat_layout_id = ?", layoutID).
		Order("order_index ASC, created_at ASC").
		Find(&colModels).Error; err != nil {

		r.logger.Error(
			"seat_columns_find_by_layout_id_failed",
			"seat_layout_id", layoutID,
			"error", err,
		)
		return nil, err
	}

	cols := make([]domain.SeatColumn, 0, len(colModels))
	for i := range colModels {
		cols = append(cols, *toSeatColumnDomain(&colModels[i]))
	}

	return cols, nil
}

func (r *SeatColumnRepository) FindByID(id uuid.UUID) (*domain.SeatColumn, error) {
	var colModel models.SeatColumnModel

	if err := r.db.First(&colModel, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSeatColumnNotFound
		}

		r.logger.Error(
			"seat_column_find_by_id_failed",
			"column_id", id,
			"error", err,
		)
		return nil, err
	}

	return toSeatColumnDomain(&colModel), nil
}

func toSeatColumnModel(col *domain.SeatColumn) *models.SeatColumnModel {
	return &models.SeatColumnModel{
		ID:           col.ID,
		SeatLayoutID: col.SeatLayoutID,
		Label:        col.Label,
		OrderIndex:   col.OrderIndex,
		CreatedAt:    col.CreatedAt,
		UpdatedAt:    col.UpdatedAt,
	}
}

func toSeatColumnDomain(colModel *models.SeatColumnModel) *domain.SeatColumn {
	return &domain.SeatColumn{
		ID:           colModel.ID,
		SeatLayoutID: colModel.SeatLayoutID,
		Label:        colModel.Label,
		OrderIndex:   colModel.OrderIndex,
		CreatedAt:    colModel.CreatedAt,
		UpdatedAt:    colModel.UpdatedAt,
	}
}

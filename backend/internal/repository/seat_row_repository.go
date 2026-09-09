package repository

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeatRowRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatRowRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *SeatRowRepository {
	return &SeatRowRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SeatRowRepository) Create(row *domain.SeatRow) error {
	rowModel := toSeatRowModel(row)

	if err := r.db.Create(rowModel).Error; err != nil {
		r.logger.Error(
			"seat_row_create_failed",
			"section_id", row.SectionID,
			"error", err,
		)

		return err
	}

	*row = *toSeatRowDomain(rowModel)

	r.logger.Info(
		"seat_row_created",
		"row_id", row.ID,
		"section_id", row.SectionID,
	)

	return nil
}

func (r *SeatRowRepository) FindBySectionID(
	sectionID uuid.UUID,
) ([]domain.SeatRow, error) {
	var rowModels []models.SeatRowModel

	if err := r.db.
		Where("section_id = ?", sectionID).
		Find(&rowModels).Error; err != nil {

		r.logger.Error(
			"seat_rows_find_by_section_id_failed",
			"section_id", sectionID,
			"error", err,
		)

		return nil, err
	}

	rows := make([]domain.SeatRow, 0, len(rowModels))

	for i := range rowModels {
		rows = append(rows, *toSeatRowDomain(&rowModels[i]))
	}

	return rows, nil
}

func toSeatRowModel(row *domain.SeatRow) *models.SeatRowModel {
	return &models.SeatRowModel{
		ID:        row.ID,
		SectionID: row.SectionID,
		RowName:   row.RowName,
		CreatedAt: row.CreatedAt,
	}
}

func toSeatRowDomain(rowModel *models.SeatRowModel) *domain.SeatRow {
	return &domain.SeatRow{
		ID:        rowModel.ID,
		SectionID: rowModel.SectionID,
		RowName:   rowModel.RowName,
		CreatedAt: rowModel.CreatedAt,
	}
}

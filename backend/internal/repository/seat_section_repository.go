package repository

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeatSectionRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatSectionRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *SeatSectionRepository {
	return &SeatSectionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SeatSectionRepository) Create(section *domain.SeatSection) error {
	sectionModel := toSeatSectionModel(section)

	if err := r.db.Create(sectionModel).Error; err != nil {
		r.logger.Error(
			"seat_section_create_failed",
			"seat_layout_id", section.SeatLayoutID,
			"error", err,
		)

		return err
	}

	*section = *toSeatSectionDomain(sectionModel)

	r.logger.Info(
		"seat_section_created",
		"section_id", section.ID,
		"seat_layout_id", section.SeatLayoutID,
	)

	return nil
}

func (r *SeatSectionRepository) FindBySeatLayoutID(
	seatLayoutID uuid.UUID,
) ([]domain.SeatSection, error) {
	var sectionModels []models.SeatSectionModel

	if err := r.db.
		Where("seat_layout_id = ?", seatLayoutID).
		Find(&sectionModels).Error; err != nil {

		r.logger.Error(
			"seat_sections_find_by_layout_id_failed",
			"seat_layout_id", seatLayoutID,
			"error", err,
		)

		return nil, err
	}

	sections := make([]domain.SeatSection, 0, len(sectionModels))

	for i := range sectionModels {
		sections = append(
			sections,
			*toSeatSectionDomain(&sectionModels[i]),
		)
	}

	return sections, nil
}

func toSeatSectionModel(
	section *domain.SeatSection,
) *models.SeatSectionModel {
	return &models.SeatSectionModel{
		ID:           section.ID,
		SeatLayoutID: section.SeatLayoutID,
		Name:         section.Name,
		Price:        section.Price,
		CreatedAt:    section.CreatedAt,
		UpdatedAt:    section.UpdatedAt,
	}
}

func toSeatSectionDomain(
	sectionModel *models.SeatSectionModel,
) *domain.SeatSection {
	return &domain.SeatSection{
		ID:           sectionModel.ID,
		SeatLayoutID: sectionModel.SeatLayoutID,
		Name:         sectionModel.Name,
		Price:        sectionModel.Price,
		CreatedAt:    sectionModel.CreatedAt,
		UpdatedAt:    sectionModel.UpdatedAt,
	}
}

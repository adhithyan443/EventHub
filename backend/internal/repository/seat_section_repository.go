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

func (r *SeatSectionRepository) FindWithStatsBySeatLayoutID(
	seatLayoutID uuid.UUID,
) ([]domain.SeatSectionWithStats, error) {
	type sectionStatRow struct {
		ID                uuid.UUID `gorm:"column:id"`
		SeatLayoutID      uuid.UUID `gorm:"column:seat_layout_id"`
		Name              string    `gorm:"column:name"`
		Price             float64   `gorm:"column:price"`
		TotalCapacity     int       `gorm:"column:total_capacity"`
		AvailableQuantity int       `gorm:"column:available_quantity"`
		SoldQuantity      int       `gorm:"column:sold_quantity"`
	}

	var rows []sectionStatRow

	query := `
		SELECT 
			ss.id,
			ss.seat_layout_id,
			ss.name,
			ss.price,
			COALESCE(COUNT(CASE WHEN s.status != 'DISABLED' THEN 1 END), 0) AS total_capacity,
			COALESCE(COUNT(CASE WHEN s.status = 'AVAILABLE' THEN 1 END), 0) AS available_quantity,
			COALESCE(COUNT(CASE WHEN s.status = 'BOOKED' THEN 1 END), 0) AS sold_quantity
		FROM seat_sections ss
		LEFT JOIN seat_rows sr ON sr.section_id = ss.id
		LEFT JOIN seats s ON s.row_id = sr.id
		WHERE ss.seat_layout_id = ?
		GROUP BY ss.id, ss.seat_layout_id, ss.name, ss.price, ss.created_at
		ORDER BY ss.created_at ASC
	`

	if err := r.db.Raw(query, seatLayoutID).Scan(&rows).Error; err != nil {
		r.logger.Error(
			"seat_sections_find_with_stats_failed",
			"seat_layout_id", seatLayoutID,
			"error", err,
		)
		return nil, err
	}

	stats := make([]domain.SeatSectionWithStats, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, domain.SeatSectionWithStats{
			ID:                row.ID,
			SeatLayoutID:      row.SeatLayoutID,
			Name:              row.Name,
			Price:             row.Price,
			TotalCapacity:     row.TotalCapacity,
			AvailableQuantity: row.AvailableQuantity,
			SoldQuantity:      row.SoldQuantity,
		})
	}

	return stats, nil
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

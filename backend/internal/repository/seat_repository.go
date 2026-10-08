package repository

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeatRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSeatRepository(
	db *gorm.DB,
	logger *slog.Logger,
) *SeatRepository {
	return &SeatRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SeatRepository) Create(seat *domain.Seat) error {
	seatModel := toSeatModel(seat)

	if err := r.db.Create(seatModel).Error; err != nil {
		r.logger.Error(
			"seat_create_failed",
			"row_id", seat.RowID,
			"seat_number", seat.SeatNumber,
			"error", err,
		)

		return err
	}

	*seat = *toSeatDomain(seatModel)

	r.logger.Info(
		"seat_created",
		"seat_id", seat.ID,
		"row_id", seat.RowID,
		"seat_number", seat.SeatNumber,
	)

	return nil
}

func (r *SeatRepository) Update(seat *domain.Seat) error {
	seatModel := toSeatModel(seat)

	if err := r.db.Save(seatModel).Error; err != nil {
		r.logger.Error(
			"seat_update_failed",
			"seat_id", seat.ID,
			"row_id", seat.RowID,
			"error", err,
		)
		return err
	}

	*seat = *toSeatDomain(seatModel)

	r.logger.Info(
		"seat_updated",
		"seat_id", seat.ID,
		"row_id", seat.RowID,
	)

	return nil
}

func (r *SeatRepository) Delete(id uuid.UUID) error {
	if err := r.db.Where("id = ?", id).Delete(&models.SeatModel{}).Error; err != nil {
		r.logger.Error(
			"seat_delete_failed",
			"seat_id", id,
			"error", err,
		)
		return err
	}

	r.logger.Info(
		"seat_deleted",
		"seat_id", id,
	)

	return nil
}

func (r *SeatRepository) DeleteByRowID(rowID uuid.UUID) error {
	if err := r.db.Where("row_id = ?", rowID).Delete(&models.SeatModel{}).Error; err != nil {
		r.logger.Error(
			"seats_delete_by_row_failed",
			"row_id", rowID,
			"error", err,
		)
		return err
	}

	r.logger.Info(
		"seats_deleted_by_row",
		"row_id", rowID,
	)

	return nil
}

func (r *SeatRepository) CountBookedByLayoutID(layoutID uuid.UUID) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(s.id)
		FROM seats s
		JOIN seat_rows sr ON sr.id = s.row_id
		JOIN seat_sections ss ON ss.id = sr.section_id
		WHERE ss.seat_layout_id = ? AND s.status IN (?, ?)
	`

	if err := r.db.Raw(query, layoutID, domain.SeatStatusBooked, domain.SeatStatusReserved).Scan(&count).Error; err != nil {
		r.logger.Error(
			"count_booked_seats_by_layout_failed",
			"seat_layout_id", layoutID,
			"error", err,
		)
		return 0, err
	}

	return count, nil
}

func (r *SeatRepository) FindByRowID(
	rowID uuid.UUID,
) ([]domain.Seat, error) {
	var seatModels []models.SeatModel

	if err := r.db.
		Where("row_id = ?", rowID).
		Order("seat_number ASC").
		Find(&seatModels).Error; err != nil {

		r.logger.Error(
			"seats_find_by_row_id_failed",
			"row_id", rowID,
			"error", err,
		)

		return nil, err
	}

	seats := make([]domain.Seat, 0, len(seatModels))

	for i := range seatModels {
		seats = append(seats, *toSeatDomain(&seatModels[i]))
	}

	return seats, nil
}

func toSeatModel(seat *domain.Seat) *models.SeatModel {
	return &models.SeatModel{
		ID:         seat.ID,
		RowID:      seat.RowID,
		SeatNumber: seat.SeatNumber,
		Status:     seat.Status,
		CreatedAt:  seat.CreatedAt,
	}
}

func toSeatDomain(seatModel *models.SeatModel) *domain.Seat {
	return &domain.Seat{
		ID:         seatModel.ID,
		RowID:      seatModel.RowID,
		SeatNumber: seatModel.SeatNumber,
		Status:     seatModel.Status,
		CreatedAt:  seatModel.CreatedAt,
	}
}

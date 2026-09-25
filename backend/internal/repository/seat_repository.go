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

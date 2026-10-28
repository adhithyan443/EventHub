package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type bookingRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewBookingRepository(db *gorm.DB, logger *slog.Logger) domain.BookingRepository {
	return &bookingRepository{
		db:     db,
		logger: logger,
	}
}

// Create persists a booking and its items.
// It relies on the surrounding transaction manager for atomicity.
func (r *bookingRepository) Create(
	ctx context.Context,
	booking *domain.Booking,
	items []domain.BookingItem,
) error {
	if booking == nil || len(items) == 0 {
		return errors.New("booking and at least one booking item are required")
	}

	bookingModel := bookingToModel(booking)

	itemModels := make([]models.BookingItemModel, 0, len(items))
	for _, item := range items {
		itemModels = append(itemModels, models.BookingItemModel{
			ID:           item.ID,
			BookingID:    booking.ID,
			SeatID:       item.SeatID,
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			TotalPrice:   item.TotalPrice,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}

	db := r.db.WithContext(ctx)

	if err := db.Create(bookingModel).Error; err != nil {
		if isDuplicateReservationBooking(err) {
			r.logger.WarnContext(ctx, "duplicate reservation booking rejected",
				slog.String("user_id", booking.UserID.String()),
				slog.String("error", err.Error()),
			)
			return domain.ErrDuplicateReservationBooking
		}

		r.logger.ErrorContext(ctx, "failed to create booking",
			slog.String("booking_id", booking.ID.String()),
			slog.String("booking_number", booking.BookingNumber),
			slog.String("user_id", booking.UserID.String()),
			slog.String("event_id", booking.EventID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	if err := db.Create(&itemModels).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create booking items",
			slog.String("booking_id", booking.ID.String()),
			slog.Int("item_count", len(itemModels)),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (r *bookingRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Booking, error) {
	var bookingModel models.BookingModel

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&bookingModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find booking by ID",
			slog.String("booking_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return bookingFromModel(&bookingModel), nil
}

func (r *bookingRepository) FindByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Booking, error) {
	var bookingModel models.BookingModel

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&bookingModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock booking by ID",
			slog.String("booking_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return bookingFromModel(&bookingModel), nil
}

func (r *bookingRepository) FindByBookingNumber(
	ctx context.Context,
	bookingNumber string,
) (*domain.Booking, error) {
	var bookingModel models.BookingModel

	err := r.db.WithContext(ctx).
		Where("booking_number = ?", bookingNumber).
		First(&bookingModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find booking by booking number",
			slog.String("booking_number", bookingNumber),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return bookingFromModel(&bookingModel), nil
}

func (r *bookingRepository) FindByReservationID(
	ctx context.Context,
	reservationID uuid.UUID,
) (*domain.Booking, error) {
	var bookingModel models.BookingModel

	err := r.db.WithContext(ctx).
		Where("reservation_id = ?", reservationID).
		First(&bookingModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBookingNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find booking by reservation ID",
			slog.String("reservation_id", reservationID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return bookingFromModel(&bookingModel), nil
}

func (r *bookingRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.BookingModel{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update booking status",
			slog.String("booking_id", id.String()),
			slog.String("error", result.Error.Error()),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrBookingNotFound
	}

	r.logger.InfoContext(ctx, "booking status updated",
		slog.String("booking_id", id.String()),
		slog.String("status", status),
	)

	return nil
}

func (r *bookingRepository) ConfirmPending(
	ctx context.Context,
	id uuid.UUID,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.BookingModel{}).
		Where(
			"id = ? AND status = ?",
			id,
			domain.BookingStatusPending,
		).
		Update("status", domain.BookingStatusConfirmed)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to confirm pending booking",
			slog.String("booking_id", id.String()),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrBookingNotFound
	}

	r.logger.InfoContext(ctx, "booking confirmed",
		slog.String("booking_id", id.String()),
	)

	return nil
}

func (r *bookingRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Booking, error) {
	var bookingModels []models.BookingModel

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bookingModels).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find bookings by user ID",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	bookings := make([]domain.Booking, 0, len(bookingModels))
	for _, m := range bookingModels {
		bookings = append(bookings, *bookingFromModel(&m))
	}

	return bookings, nil
}

func (r *bookingRepository) FindWithItemsByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Booking, []domain.BookingItem, error) {
	booking, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	items, err := r.FindItemsByBookingID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if items == nil {
		items = []domain.BookingItem{}
	}

	return booking, items, nil
}

func (r *bookingRepository) FindItemsByBookingID(
	ctx context.Context,
	bookingID uuid.UUID,
) ([]domain.BookingItem, error) {
	var itemModels []models.BookingItemModel

	if err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Find(&itemModels).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to find booking items",
			slog.String("booking_id", bookingID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	items := make([]domain.BookingItem, 0, len(itemModels))
	for _, item := range itemModels {
		items = append(items, domain.BookingItem{
			ID:           item.ID,
			BookingID:    item.BookingID,
			SeatID:       item.SeatID,
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			TotalPrice:   item.TotalPrice,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}

	return items, nil
}

func (r *bookingRepository) FindItemsByBookingIDs(
	ctx context.Context,
	bookingIDs []uuid.UUID,
) (map[uuid.UUID][]domain.BookingItem, error) {
	result := make(map[uuid.UUID][]domain.BookingItem)
	if len(bookingIDs) == 0 {
		return result, nil
	}

	var itemModels []models.BookingItemModel
	if err := r.db.WithContext(ctx).
		Where("booking_id IN ?", bookingIDs).
		Find(&itemModels).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to find booking items by booking IDs",
			slog.Int("booking_count", len(bookingIDs)),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	for _, item := range itemModels {
		result[item.BookingID] = append(result[item.BookingID], domain.BookingItem{
			ID:           item.ID,
			BookingID:    item.BookingID,
			SeatID:       item.SeatID,
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			TotalPrice:   item.TotalPrice,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}

	return result, nil
}

func bookingToModel(booking *domain.Booking) *models.BookingModel {
	return &models.BookingModel{
		ID:                        booking.ID,
		BookingNumber:             booking.BookingNumber,
		UserID:                    booking.UserID,
		EventID:                   booking.EventID,
		ReservationID:             booking.ReservationID,
		Subtotal:                  booking.Subtotal,
		PlatformFee:               booking.PlatformFee,
		Tax:                       booking.Tax,
		Discount:                  booking.Discount,
		Total:                     booking.Total,
		CancellationAllowed:       booking.CancellationAllowed,
		CancellationDeadlineHours: booking.CancellationDeadlineHours,
		RefundPolicy:              booking.RefundPolicy,
		RefundPercentage:          booking.RefundPercentage,
		Status:                    booking.Status,
		CreatedAt:                 booking.CreatedAt,
		UpdatedAt:                 booking.UpdatedAt,
	}
}

func bookingFromModel(model *models.BookingModel) *domain.Booking {
	return &domain.Booking{
		ID:                        model.ID,
		BookingNumber:             model.BookingNumber,
		UserID:                    model.UserID,
		EventID:                   model.EventID,
		ReservationID:             model.ReservationID,
		Subtotal:                  model.Subtotal,
		PlatformFee:               model.PlatformFee,
		Tax:                       model.Tax,
		Discount:                  model.Discount,
		Total:                     model.Total,
		CancellationAllowed:       model.CancellationAllowed,
		CancellationDeadlineHours: model.CancellationDeadlineHours,
		RefundPolicy:              model.RefundPolicy,
		RefundPercentage:          model.RefundPercentage,
		Status:                    model.Status,
		CreatedAt:                 model.CreatedAt,
		UpdatedAt:                 model.UpdatedAt,
	}
}

func isDuplicateReservationBooking(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" && //unique constraint violation code 23505
		pgErr.ConstraintName == "uq_bookings_reservation_id"
}

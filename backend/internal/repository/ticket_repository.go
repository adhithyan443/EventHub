package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ticketRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTicketRepository(
	db *gorm.DB,
	logger *slog.Logger,
) domain.TicketRepository {
	return &ticketRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ticketRepository) Create(
	ctx context.Context,
	ticket *domain.Ticket,
) error {
	model := toTicketModel(ticket)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create ticket",
			slog.String("booking_id", ticket.BookingID.String()),
			slog.String("event_id", ticket.EventID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	ticket.ID = model.ID
	ticket.CreatedAt = model.CreatedAt
	ticket.UpdatedAt = model.UpdatedAt

	r.logger.InfoContext(ctx, "ticket created",
		slog.String("ticket_id", ticket.ID.String()),
		slog.String("booking_id", ticket.BookingID.String()),
	)

	return nil
}

func (r *ticketRepository) CreateMany(
	ctx context.Context,
	tickets []domain.Ticket,
) error {
	if len(tickets) == 0 {
		return nil
	}

	modelsList := make([]models.TicketModel, len(tickets))

	for i := range tickets {
		modelsList[i] = *toTicketModel(&tickets[i])
	}

	if err := r.db.WithContext(ctx).Create(&modelsList).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create tickets",
			slog.Int("ticket_count", len(tickets)),
			slog.String("error", err.Error()),
		)
		return err
	}

	for i := range tickets {
		tickets[i].ID = modelsList[i].ID
		tickets[i].CreatedAt = modelsList[i].CreatedAt
		tickets[i].UpdatedAt = modelsList[i].UpdatedAt
	}

	r.logger.InfoContext(ctx, "tickets created",
		slog.Int("ticket_count", len(tickets)),
	)

	return nil
}

func (r *ticketRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Ticket, error) {
	var model models.TicketModel

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrTicketNotFound
	}

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find ticket",
			slog.String("ticket_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return toTicketDomain(&model), nil
}

func (r *ticketRepository) FindByCode(
	ctx context.Context,
	code string,
) (*domain.Ticket, error) {
	var model models.TicketModel

	err := r.db.WithContext(ctx).
		Where("ticket_code = ?", code).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrTicketNotFound
	}

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find ticket by code",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return toTicketDomain(&model), nil
}

func (r *ticketRepository) FindByBookingID(
	ctx context.Context,
	bookingID uuid.UUID,
) ([]domain.Ticket, error) {
	var ticketModels []models.TicketModel

	if err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("issued_at ASC").
		Find(&ticketModels).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to find tickets by booking",
			slog.String("booking_id", bookingID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	tickets := make([]domain.Ticket, len(ticketModels))

	for i := range ticketModels {
		tickets[i] = *toTicketDomain(&ticketModels[i])
	}

	return tickets, nil
}

func (r *ticketRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Ticket, error) {
	var ticketModels []models.TicketModel

	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Find(&ticketModels).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to find tickets by user",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	tickets := make([]domain.Ticket, len(ticketModels))

	for i := range ticketModels {
		tickets[i] = *toTicketDomain(&ticketModels[i])
	}

	return tickets, nil
}

func (r *ticketRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.TicketModel{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update ticket status",
			slog.String("ticket_id", id.String()),
			slog.String("error", result.Error.Error()),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrTicketNotFound
	}

	r.logger.InfoContext(ctx, "ticket status updated",
		slog.String("ticket_id", id.String()),
		slog.String("status", status),
	)

	return nil
}

func toTicketModel(ticket *domain.Ticket) *models.TicketModel {
	return &models.TicketModel{
		ID:              ticket.ID,
		TicketCode:      ticket.TicketCode,
		BookingID:       ticket.BookingID,
		BookingItemID:   ticket.BookingItemID,
		UserID:          ticket.UserID,
		EventID:         ticket.EventID,
		TicketTypeID:    ticket.TicketTypeID,
		SeatID:          ticket.SeatID,
		Price:           ticket.Price,
		Status:          ticket.Status,
		AdmissionNumber: ticket.AdmissionNumber,
		IssuedAt:        ticket.IssuedAt,
		UsedAt:          ticket.UsedAt,
	}
}

func toTicketDomain(model *models.TicketModel) *domain.Ticket {
	return &domain.Ticket{
		ID:              model.ID,
		TicketCode:      model.TicketCode,
		BookingID:       model.BookingID,
		BookingItemID:   model.BookingItemID,
		UserID:          model.UserID,
		EventID:         model.EventID,
		TicketTypeID:    model.TicketTypeID,
		SeatID:          model.SeatID,
		Price:           model.Price,
		Status:          model.Status,
		AdmissionNumber: model.AdmissionNumber,
		IssuedAt:        model.IssuedAt,
		UsedAt:          model.UsedAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

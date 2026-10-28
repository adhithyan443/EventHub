package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type refundRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewRefundRepository(db *gorm.DB, logger *slog.Logger) domain.RefundRepository {
	return &refundRepository{
		db:     db,
		logger: logger,
	}
}

func (r *refundRepository) Create(ctx context.Context, refund *domain.Refund) error {
	if refund == nil {
		return errors.New("refund cannot be nil")
	}

	model := toRefundModel(refund)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create refund",
			slog.String("refund_id", refund.ID.String()),
			slog.String("booking_id", refund.BookingID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (r *refundRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	var model models.RefundModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRefundNotFound
		}
		r.logger.ErrorContext(ctx, "failed to find refund by ID",
			slog.String("refund_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return toRefundDomain(&model), nil
}

func (r *refundRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Refund, error) {
	var model models.RefundModel
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRefundNotFound
		}
		r.logger.ErrorContext(ctx, "failed to lock refund by ID",
			slog.String("refund_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return toRefundDomain(&model), nil
}

func (r *refundRepository) FindByBookingID(ctx context.Context, bookingID uuid.UUID) ([]domain.Refund, error) {
	var refundModels []models.RefundModel
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at ASC").
		Find(&refundModels).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find refunds by booking ID",
			slog.String("booking_id", bookingID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	refunds := make([]domain.Refund, len(refundModels))
	for i := range refundModels {
		refunds[i] = *toRefundDomain(&refundModels[i])
	}

	return refunds, nil
}

func (r *refundRepository) Update(ctx context.Context, refund *domain.Refund) error {
	if refund == nil {
		return errors.New("refund cannot be nil")
	}

	model := toRefundModel(refund)
	result := r.db.WithContext(ctx).
		Model(&models.RefundModel{}).
		Where("id = ?", refund.ID).
		Updates(map[string]interface{}{
			"gateway_refund_id": model.GatewayRefundID,
			"status":            model.Status,
			"processed_at":      model.ProcessedAt,
			"updated_at":        model.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update refund",
			slog.String("refund_id", refund.ID.String()),
			slog.String("error", result.Error.Error()),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrRefundNotFound
	}

	return nil
}

func toRefundModel(r *domain.Refund) *models.RefundModel {
	return &models.RefundModel{
		ID:              r.ID,
		BookingID:       r.BookingID,
		PaymentID:       r.PaymentID,
		GatewayRefundID: r.GatewayRefundID,
		Amount:          r.Amount,
		Status:          r.Status,
		Reason:          r.Reason,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		ProcessedAt:     r.ProcessedAt,
	}
}

func toRefundDomain(m *models.RefundModel) *domain.Refund {
	return &domain.Refund{
		ID:              m.ID,
		BookingID:       m.BookingID,
		PaymentID:       m.PaymentID,
		GatewayRefundID: m.GatewayRefundID,
		Amount:          m.Amount,
		Status:          m.Status,
		Reason:          m.Reason,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		ProcessedAt:     m.ProcessedAt,
	}
}

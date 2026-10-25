package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type paymentRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewPaymentRepository(db *gorm.DB, logger *slog.Logger) domain.PaymentRepository {
	return &paymentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *paymentRepository) Create(
	ctx context.Context,
	payment *domain.Payment,
) error {
	model := toPaymentModel(payment)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create payment",
			slog.String("booking_id", payment.BookingID),
			slog.Any("error", err),
		)
		return err
	}

	payment.ID = model.ID
	payment.CreatedAt = model.CreatedAt
	payment.UpdatedAt = model.UpdatedAt

	return nil
}

func (r *paymentRepository) FindByID(
	ctx context.Context,
	id string,
) (*domain.Payment, error) {
	var model models.PaymentModel

	err := r.db.WithContext(ctx).
		First(&model, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find payment",
			slog.String("payment_id", id),
			slog.Any("error", err),
		)
		return nil, err
	}

	return toPaymentDomain(&model), nil
}

func (r *paymentRepository) FindByBookingID(
	ctx context.Context,
	bookingID string,
) ([]domain.Payment, error) {
	var paymentModels []models.PaymentModel

	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Find(&paymentModels).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find payments by booking",
			slog.String("booking_id", bookingID),
			slog.Any("error", err),
		)
		return nil, err
	}

	payments := make([]domain.Payment, 0, len(paymentModels))
	for i := range paymentModels {
		payments = append(payments, *toPaymentDomain(&paymentModels[i]))
	}

	return payments, nil
}

func (r *paymentRepository) Update(
	ctx context.Context,
	payment *domain.Payment,
) error {
	model := toPaymentModel(payment)

	result := r.db.WithContext(ctx).
		Model(&models.PaymentModel{}).
		Where("id = ?", payment.ID).
		Updates(map[string]interface{}{
			"transaction_id":    model.TransactionID,
			"provider":          model.Provider,
			"provider_order_id": model.ProviderOrderID,
			"payment_method":    model.PaymentMethod,
			"amount":            model.Amount,
			"status":            model.Status,
			"paid_at":           model.PaidAt,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update payment",
			slog.String("payment_id", payment.ID),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// MarkSuccessful atomically transitions a pending payment to success.
// It prevents another request from overwriting an already completed payment.
func (r *paymentRepository) MarkSuccessful(
	ctx context.Context,
	paymentID string,
	transactionID string,
	paidAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.PaymentModel{}).
		Where(
			"id = ? AND status = ?",
			paymentID,
			string(domain.PaymentStatusPending),
		).
		Updates(map[string]interface{}{
			"transaction_id": transactionID,
			"status":         string(domain.PaymentStatusSuccess),
			"paid_at":        paidAt,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to mark payment successful",
			slog.String("payment_id", paymentID),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		var count int64
		err := r.db.WithContext(ctx).
			Model(&models.PaymentModel{}).
			Where("id = ?", paymentID).
			Count(&count).Error
		if err != nil {
			return err
		}

		if count == 0 {
			return domain.ErrPaymentNotFound
		}

		return domain.ErrPaymentAlreadyComplete
	}

	return nil
}

// MarkFailed conditionally transitions a pending payment to failed.
// It will not overwrite a payment that is already marked successful or failed.
func (r *paymentRepository) MarkFailed(
	ctx context.Context,
	paymentID string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.PaymentModel{}).
		Where(
			"id = ? AND status = ?",
			paymentID,
			string(domain.PaymentStatusPending),
		).
		Update("status", string(domain.PaymentStatusFailed))

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to mark payment failed",
			slog.String("payment_id", paymentID),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		var count int64
		err := r.db.WithContext(ctx).
			Model(&models.PaymentModel{}).
			Where("id = ?", paymentID).
			Count(&count).Error
		if err != nil {
			return err
		}

		if count == 0 {
			return domain.ErrPaymentNotFound
		}

		// The payment exists but is not pending (already success or failed).
		// Safely do nothing to protect existing state.
		return nil
	}

	return nil
}

// MarkRefundRequired conditionally transitions a pending payment to REFUND_REQUIRED,
// persisting the gateway transaction ID and paid timestamp for refund processing.
func (r *paymentRepository) MarkRefundRequired(
	ctx context.Context,
	paymentID string,
	transactionID string,
	paidAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.PaymentModel{}).
		Where(
			"id = ? AND status = ?",
			paymentID,
			string(domain.PaymentStatusPending),
		).
		Updates(map[string]interface{}{
			"transaction_id": transactionID,
			"status":         string(domain.PaymentStatusRefundRequired),
			"paid_at":        paidAt,
		})

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to mark payment refund required",
			slog.String("payment_id", paymentID),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		var count int64
		err := r.db.WithContext(ctx).
			Model(&models.PaymentModel{}).
			Where("id = ?", paymentID).
			Count(&count).Error
		if err != nil {
			return err
		}

		if count == 0 {
			return domain.ErrPaymentNotFound
		}

		return domain.ErrPaymentAlreadyComplete
	}

	return nil
}

func (r *paymentRepository) FindByProviderOrderID(
	ctx context.Context,
	providerOrderID string,
) (*domain.Payment, error) {
	var model models.PaymentModel

	err := r.db.WithContext(ctx).
		Where("provider_order_id = ?", providerOrderID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}

		r.logger.ErrorContext(ctx, "failed to find payment by provider order ID",
			slog.Any("error", err),
		)
		return nil, err
	}

	return toPaymentDomain(&model), nil
}

func (r *paymentRepository) FindByProviderOrderIDForUpdate(
	ctx context.Context,
	providerOrderID string,
) (*domain.Payment, error) {
	var model models.PaymentModel

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("provider_order_id = ?", providerOrderID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}

		r.logger.ErrorContext(ctx, "failed to lock payment by provider order ID",
			slog.Any("error", err),
		)
		return nil, err
	}

	return toPaymentDomain(&model), nil
}

func toPaymentModel(payment *domain.Payment) *models.PaymentModel {
	return &models.PaymentModel{
		ID:              payment.ID,
		BookingID:       payment.BookingID,
		TransactionID:   payment.TransactionID,
		Provider:        string(payment.Provider),
		ProviderOrderID: payment.ProviderOrderID,
		PaymentMethod:   string(payment.Method),
		Amount:          payment.Amount,
		Status:          string(payment.Status),
		PaidAt:          payment.PaidAt,
		CreatedAt:       payment.CreatedAt,
		UpdatedAt:       payment.UpdatedAt,
	}
}

func toPaymentDomain(model *models.PaymentModel) *domain.Payment {
	return &domain.Payment{
		ID:              model.ID,
		BookingID:       model.BookingID,
		TransactionID:   model.TransactionID,
		Provider:        domain.PaymentProvider(model.Provider),
		ProviderOrderID: model.ProviderOrderID,
		Method:          domain.PaymentMethod(model.PaymentMethod),
		Amount:          model.Amount,
		Status:          domain.PaymentStatus(model.Status),
		PaidAt:          model.PaidAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

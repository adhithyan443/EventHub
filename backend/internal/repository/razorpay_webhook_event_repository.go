package repository

import (
	"context"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type razorpayWebhookEventRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewRazorpayWebhookEventRepository(
	db *gorm.DB,
	logger *slog.Logger,
) domain.RazorpayWebhookEventRepository {
	return &razorpayWebhookEventRepository{
		db:     db,
		logger: logger,
	}
}

func (r *razorpayWebhookEventRepository) Create(
	ctx context.Context,
	eventID string,
	eventType string,
) error {
	event := &models.RazorpayWebhookEventModel{
		EventID:   eventID,
		EventType: eventType,
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(event)

	if result.Error != nil {
		r.logger.ErrorContext(
			ctx,
			"failed to record Razorpay webhook event",
			slog.String("event_type", eventType),
			slog.Any("error", result.Error),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrDuplicateWebhookEvent
	}

	return nil
}

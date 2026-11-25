package repository

import (
	"context"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"gorm.io/gorm"
)

type ticketScanRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTicketScanRepository(
	db *gorm.DB,
	logger *slog.Logger,
) domain.TicketScanRepository {
	return &ticketScanRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ticketScanRepository) Create(
	ctx context.Context,
	scan *domain.TicketScan,
) error {
	model := &models.TicketScanModel{
		ID:        scan.ID,
		TicketID:  scan.TicketID,
		ScannedBy: scan.ScannedBy,
		Status:    scan.Status,
		ScannedAt: scan.ScannedAt,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create ticket scan",
			slog.String("ticket_id", scan.TicketID.String()),
			slog.String("id", scan.ID.String()),
			slog.String("error", err.Error()),
		)

		return err
	}

	r.logger.InfoContext(ctx, "ticket scan recorded",
		slog.String("ticket_id", scan.TicketID.String()),
		slog.String("id", scan.ID.String()),
	)

	return nil
}

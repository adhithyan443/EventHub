package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type ReservationExpiryUsecase interface {
	ExpireReservation(
		ctx context.Context,
		reservationID uuid.UUID,
	) error

	ProcessExpiredReservations(
		ctx context.Context,
		limit int,
	) error
}

type ReservationExpiryWorker struct {
	usecase  ReservationExpiryUsecase
	logger   *slog.Logger
	interval time.Duration
}

func NewReservationExpiryWorker(
	usecase ReservationExpiryUsecase,
	logger *slog.Logger,
	interval time.Duration,
) *ReservationExpiryWorker {
	return &ReservationExpiryWorker{
		usecase:  usecase,
		logger:   logger,
		interval: interval,
	}
}

func (w *ReservationExpiryWorker) Run(ctx context.Context) {
	if w.interval <= 0 {
		w.logger.ErrorContext(
			ctx,
			"reservation expiry worker has invalid interval",
		)
		return
	}

	ticker := time.NewTicker(w.interval) // it will give config interval signal
	defer ticker.Stop()

	w.logger.InfoContext(
		ctx,
		"reservation expiry worker started",
		slog.Duration("interval", w.interval),
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(
				ctx,
				"reservation expiry worker stopped",
			)
			return

		case <-ticker.C:
			if err := w.usecase.ProcessExpiredReservations(ctx, 100); err != nil {
				w.logger.ErrorContext(
					ctx,
					"reservation expiry batch processing failed",
					slog.String("error", err.Error()),
				)
			}

			// Expired reservations processing will be added next.
		}
	}
}

package worker

import (
	"context"
	"log/slog"
	"time"
)

type RefundReconciliationUsecase interface {
	ReconcileStuckRefunds(
		ctx context.Context,
		olderThan time.Duration,
		limit int,
	) (int, error)
}

type RefundReconciliationWorker struct {
	usecase   RefundReconciliationUsecase
	logger    *slog.Logger
	interval  time.Duration
	olderThan time.Duration
}

func NewRefundReconciliationWorker(
	usecase RefundReconciliationUsecase,
	logger *slog.Logger,
	interval time.Duration,
	olderThan time.Duration,
) *RefundReconciliationWorker {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	if olderThan <= 0 {
		olderThan = 5 * time.Minute
	}

	return &RefundReconciliationWorker{
		usecase:   usecase,
		logger:    logger,
		interval:  interval,
		olderThan: olderThan,
	}
}

func (w *RefundReconciliationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.InfoContext(
		ctx,
		"refund reconciliation worker started",
		slog.Duration("interval", w.interval),
		slog.Duration("older_than", w.olderThan),
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(
				ctx,
				"refund reconciliation worker stopped",
			)
			return

		case <-ticker.C:
			count, err := w.usecase.ReconcileStuckRefunds(ctx, w.olderThan, 50)
			if err != nil {
				w.logger.ErrorContext(
					ctx,
					"refund reconciliation batch failed",
					slog.String("error", err.Error()),
				)
			} else if count > 0 {
				w.logger.InfoContext(
					ctx,
					"refund reconciliation batch completed",
					slog.Int("reconciled_count", count),
				)
			}
		}
	}
}

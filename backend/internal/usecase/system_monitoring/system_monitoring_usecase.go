package systemmonitoring

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

var (
	ErrInvalidLogLevel   = errors.New("invalid log level parameter")
	ErrInvalidPagination = errors.New("invalid pagination parameter")
)

type systemMonitoringUsecase struct {
	repo   domain.SystemMonitoringRepository
	logger *slog.Logger
}

// NewSystemMonitoringUsecase creates a new usecase instance for system monitoring.
func NewSystemMonitoringUsecase(
	repo domain.SystemMonitoringRepository,
	logger *slog.Logger,
) domain.SystemMonitoringUsecase {
	if logger == nil {
		logger = slog.Default()
	}
	return &systemMonitoringUsecase{
		repo:   repo,
		logger: logger,
	}
}

// GetLogs validates filter parameters and fetches paginated application logs.
func (u *systemMonitoringUsecase) GetLogs(
	ctx context.Context,
	filter domain.SystemLogFilter,
) (*domain.SystemLogListResult, error) {
	if filter.Level != "" && !strings.EqualFold(filter.Level, "ALL") {
		validLevels := map[string]bool{
			"INFO":  true,
			"WARN":  true,
			"ERROR": true,
			"DEBUG": true,
		}
		if !validLevels[strings.ToUpper(strings.TrimSpace(filter.Level))] {
			u.logger.WarnContext(ctx, "invalid log level requested", slog.String("level", filter.Level))
			return nil, ErrInvalidLogLevel
		}
	}

	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > 200 {
		return nil, ErrInvalidPagination
	}

	result, err := u.repo.GetLogs(ctx, filter)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get system logs", slog.Any("error", err))
		return nil, err
	}

	return result, nil
}

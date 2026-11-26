package systemmonitoring

import (
	"context"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

type mockSystemMonitoringRepo struct {
	logsFn func(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error)
}

func (m *mockSystemMonitoringRepo) GetLogs(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error) {
	if m.logsFn != nil {
		return m.logsFn(ctx, filter)
	}
	return &domain.SystemLogListResult{
		Logs:         []domain.SystemLogEntry{},
		TotalRecords: 0,
		Page:         filter.Page,
		Limit:        filter.Limit,
	}, nil
}

func TestSystemMonitoringUsecase_GetLogs(t *testing.T) {
	mockRepo := &mockSystemMonitoringRepo{
		logsFn: func(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error) {
			return &domain.SystemLogListResult{
				Logs: []domain.SystemLogEntry{
					{
						Timestamp: time.Now(),
						Level:     "INFO",
						Message:   "test_message",
					},
				},
				TotalRecords: 1,
				Page:         filter.Page,
				Limit:        filter.Limit,
				TotalPages:   1,
			}, nil
		},
	}

	uc := NewSystemMonitoringUsecase(mockRepo, nil)
	ctx := context.Background()

	// 1. Success case
	res, err := uc.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 50, Level: "INFO"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if res.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", res.TotalRecords)
	}

	// 2. Invalid log level
	_, err = uc.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 50, Level: "UNKNOWN_LEVEL"})
	if err != ErrInvalidLogLevel {
		t.Errorf("expected ErrInvalidLogLevel, got: %v", err)
	}

	// 3. Invalid pagination
	_, err = uc.GetLogs(ctx, domain.SystemLogFilter{Page: 0, Limit: 50})
	if err != ErrInvalidPagination {
		t.Errorf("expected ErrInvalidPagination for page 0, got: %v", err)
	}

	_, err = uc.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 500})
	if err != ErrInvalidPagination {
		t.Errorf("expected ErrInvalidPagination for limit 500, got: %v", err)
	}
}

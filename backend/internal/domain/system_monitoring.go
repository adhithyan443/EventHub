package domain

import (
	"context"
	"time"
)

// SystemLogEntry represents a structured log entry captured from the application logging system.
type SystemLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// SystemLogFilter contains query parameters for searching and paginating application logs.
type SystemLogFilter struct {
	Level  string `json:"level"`
	Search string `json:"search"`
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
}

// SystemLogListResult holds a paginated collection of log entries.
type SystemLogListResult struct {
	Logs         []SystemLogEntry `json:"logs"`
	TotalRecords int64            `json:"total_records"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
	TotalPages   int              `json:"total_pages"`
}

// SystemMonitoringRepository defines the contract for reading system logs safely.
type SystemMonitoringRepository interface {
	GetLogs(ctx context.Context, filter SystemLogFilter) (*SystemLogListResult, error)
}

// SystemMonitoringUsecase defines the business logic for admin system monitoring.
type SystemMonitoringUsecase interface {
	GetLogs(ctx context.Context, filter SystemLogFilter) (*SystemLogListResult, error)
}

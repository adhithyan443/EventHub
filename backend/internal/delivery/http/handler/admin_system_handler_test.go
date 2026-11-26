package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type mockSystemMonitoringUsecase struct {
	getLogsFn func(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error)
}

func (m *mockSystemMonitoringUsecase) GetLogs(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error) {
	if m.getLogsFn != nil {
		return m.getLogsFn(ctx, filter)
	}
	return &domain.SystemLogListResult{
		Logs:         []domain.SystemLogEntry{},
		TotalRecords: 0,
		Page:         filter.Page,
		Limit:        filter.Limit,
	}, nil
}

func TestAdminSystemHandler_GetLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockSystemMonitoringUsecase{
		getLogsFn: func(ctx context.Context, filter domain.SystemLogFilter) (*domain.SystemLogListResult, error) {
			return &domain.SystemLogListResult{
				Logs: []domain.SystemLogEntry{
					{
						Timestamp: time.Now(),
						Level:     "INFO",
						Message:   "admin_login_success",
						RequestID: "req-12345",
					},
				},
				TotalRecords: 1,
				Page:         filter.Page,
				Limit:        filter.Limit,
				TotalPages:   1,
			}, nil
		},
	}

	h := NewAdminSystemHandler(mockUC, nil)

	r := gin.New()
	r.GET("/api/v1/admin/system/logs", h.GetLogs)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/system/logs?page=1&limit=50&level=INFO", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Logs         []domain.SystemLogEntry `json:"logs"`
			TotalRecords int64                   `json:"total_records"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Data.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", resp.Data.TotalRecords)
	}
	if len(resp.Data.Logs) != 1 || resp.Data.Logs[0].Message != "admin_login_success" {
		t.Errorf("unexpected logs in response: %+v", resp.Data.Logs)
	}
}

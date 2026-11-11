package handler_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPublicEventRepo struct {
	domain.EventRepository
	capturedFilter domain.PublicEventFilter
	events         []*domain.PublicEvent
	total          int64
	err            error
}

func (m *mockPublicEventRepo) FindPublicEvents(filter domain.PublicEventFilter) ([]*domain.PublicEvent, int64, error) {
	m.capturedFilter = filter
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.events, m.total, nil
}

type mockPublicTxRepos struct {
	domain.TransactionRepositories
	eventRepo *mockPublicEventRepo
}

func (r *mockPublicTxRepos) EventRepository() domain.EventRepository {
	return r.eventRepo
}

type mockPublicTxManager struct {
	repos *mockPublicTxRepos
}

func (m *mockPublicTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func setupPublicEventRouter(repo *mockPublicEventRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	txManager := &mockPublicTxManager{
		repos: &mockPublicTxRepos{eventRepo: repo},
	}
	uc := events.NewEventUsecase(txManager, nil, slog.Default())
	h := handler.NewEventHandler(uc, slog.Default())

	r := gin.New()
	r.GET("/api/v1/events", h.ListPublicEvents)
	return r
}

func TestListPublicEvents_DefaultParams(t *testing.T) {
	repo := &mockPublicEventRepo{
		events: []*domain.PublicEvent{
			{
				ID:            uuid.New(),
				Title:         "Festival 2026",
				StartingPrice: 500,
				Status:        domain.EventStatusPublished,
				Visibility:    "PUBLIC",
			},
		},
		total: 1,
	}

	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, repo.capturedFilter.Page)
	assert.Equal(t, 12, repo.capturedFilter.Limit)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Events []map[string]interface{} `json:"events"`
			Total  int64                    `json:"total"`
			Page   int                      `json:"page"`
			Limit  int                      `json:"limit"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int64(1), resp.Data.Total)
	assert.Len(t, resp.Data.Events, 1)
}

func TestListPublicEvents_CategoryFilter(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	catID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/events?category_id=%s", catID.String()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.capturedFilter.CategoryID)
	assert.Equal(t, catID, *repo.capturedFilter.CategoryID)
}

func TestListPublicEvents_PriceFilter(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?min_price=100.5&max_price=500.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.capturedFilter.MinPrice)
	require.NotNil(t, repo.capturedFilter.MaxPrice)
	assert.Equal(t, 100.5, *repo.capturedFilter.MinPrice)
	assert.Equal(t, 500.0, *repo.capturedFilter.MaxPrice)
}

func TestListPublicEvents_DateRangeFilter(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?start_date=2026-11-15&end_date=2026-11-20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.capturedFilter.StartDate)
	require.NotNil(t, repo.capturedFilter.EndDate)
	assert.Equal(t, "2026-11-15", repo.capturedFilter.StartDate.Format("2006-01-02"))
	assert.Equal(t, "2026-11-20", repo.capturedFilter.EndDate.Format("2006-01-02"))
}

func TestListPublicEvents_SingleDateFilter(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?date=2026-11-15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.capturedFilter.EventDate)
	assert.Equal(t, "2026-11-15", repo.capturedFilter.EventDate.Format("2006-01-02"))
}

func TestListPublicEvents_LocationFilter(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?location=Kochi", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Kochi", repo.capturedFilter.City)
}

func TestListPublicEvents_CombinedFiltersAndSort(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	catID := uuid.New()
	url := fmt.Sprintf(
		"/api/v1/events?page=2&limit=20&keyword=Rock&category_id=%s&city=Bangalore&min_price=200&max_price=1000&start_date=2026-12-01&end_date=2026-12-31&sort=price_asc",
		catID.String(),
	)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, repo.capturedFilter.Page)
	assert.Equal(t, 20, repo.capturedFilter.Limit)
	assert.Equal(t, "Rock", repo.capturedFilter.Keyword)
	assert.Equal(t, "Bangalore", repo.capturedFilter.City)
	assert.Equal(t, "price_asc", repo.capturedFilter.Sort)
	require.NotNil(t, repo.capturedFilter.CategoryID)
	assert.Equal(t, catID, *repo.capturedFilter.CategoryID)
	require.NotNil(t, repo.capturedFilter.MinPrice)
	assert.Equal(t, 200.0, *repo.capturedFilter.MinPrice)
	require.NotNil(t, repo.capturedFilter.MaxPrice)
	assert.Equal(t, 1000.0, *repo.capturedFilter.MaxPrice)
	require.NotNil(t, repo.capturedFilter.StartDate)
	assert.Equal(t, "2026-12-01", repo.capturedFilter.StartDate.Format("2006-01-02"))
	require.NotNil(t, repo.capturedFilter.EndDate)
	assert.Equal(t, "2026-12-31", repo.capturedFilter.EndDate.Format("2006-01-02"))
}

func TestListPublicEvents_EmptyResults(t *testing.T) {
	repo := &mockPublicEventRepo{
		events: []*domain.PublicEvent{},
		total:  0,
	}
	r := setupPublicEventRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?keyword=NonExistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Events []map[string]interface{} `json:"events"`
			Total  int64                    `json:"total"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int64(0), resp.Data.Total)
	assert.Empty(t, resp.Data.Events)
}

func TestListPublicEvents_InvalidParameters(t *testing.T) {
	repo := &mockPublicEventRepo{}
	r := setupPublicEventRouter(repo)

	tests := []struct {
		name       string
		query      string
		expStatus  int
		expMessage string
	}{
		{
			name:       "invalid category UUID",
			query:      "?category_id=not-a-uuid",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid category id",
		},
		{
			name:       "invalid min_price string",
			query:      "?min_price=abc",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid min_price",
		},
		{
			name:       "negative min_price",
			query:      "?min_price=-10",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid min_price",
		},
		{
			name:       "invalid max_price string",
			query:      "?max_price=xyz",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid max_price",
		},
		{
			name:       "negative max_price",
			query:      "?max_price=-5",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid max_price",
		},
		{
			name:       "min_price greater than max_price",
			query:      "?min_price=500&max_price=100",
			expStatus:  http.StatusBadRequest,
			expMessage: "min_price cannot be greater than max_price",
		},
		{
			name:       "invalid start_date format",
			query:      "?start_date=15-11-2026",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid start_date",
		},
		{
			name:       "invalid end_date format",
			query:      "?end_date=bad-date",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid end_date",
		},
		{
			name:       "start_date after end_date",
			query:      "?start_date=2026-12-05&end_date=2026-11-01",
			expStatus:  http.StatusBadRequest,
			expMessage: "start_date cannot be after end_date",
		},
		{
			name:       "invalid date format",
			query:      "?date=2026/11/15",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid date",
		},
		{
			name:       "invalid page",
			query:      "?page=0",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid page",
		},
		{
			name:       "invalid limit",
			query:      "?limit=500",
			expStatus:  http.StatusBadRequest,
			expMessage: "invalid limit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/events"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expStatus, w.Code)
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, false, resp["success"])
			assert.Equal(t, tc.expMessage, resp["message"])
		})
	}
}

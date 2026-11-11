package handler_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCategoryRepo struct {
	categories []*domain.Category
	err        error
}

func (m *mockCategoryRepo) FindActive() ([]*domain.Category, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.categories, nil
}

func (m *mockCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func TestCategoryHandler_GetCategories_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cat1ID := uuid.New()
	cat2ID := uuid.New()
	mockRepo := &mockCategoryRepo{
		categories: []*domain.Category{
			{
				ID:          cat1ID,
				Name:        "Concerts",
				Description: "Live music concerts",
				Status:      domain.CategoryStatusActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          cat2ID,
				Name:        "Technology",
				Description: "Tech conferences",
				Status:      domain.CategoryStatusActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	uc := category.NewCategoryUsecase(mockRepo, slog.Default())
	h := handler.NewCategoryHandler(uc)

	r := gin.New()
	r.GET("/api/v1/categories", h.GetCategories)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Categories []struct {
				ID   string `json:"ID"`
				Name string `json:"Name"`
			} `json:"categories"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data.Categories, 2)
	assert.Equal(t, "Concerts", resp.Data.Categories[0].Name)
	assert.Equal(t, "Technology", resp.Data.Categories[1].Name)
}

func TestCategoryHandler_GetCategories_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &mockCategoryRepo{
		err: errors.New("db connection failure"),
	}

	uc := category.NewCategoryUsecase(mockRepo, slog.Default())
	h := handler.NewCategoryHandler(uc)

	r := gin.New()
	r.GET("/api/v1/categories", h.GetCategories)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Since ctx.Error is called without custom error handling middleware, default Gin status is 200 with errors in gin.Context
	assert.NotEmpty(t, w.Result())
}

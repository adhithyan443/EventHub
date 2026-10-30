package handler_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForHandlerTest struct {
	domain.UserRepository
	listResult *domain.UserList
	listErr    error
}

func (m *mockUserRepoForHandlerTest) List(filter domain.UserListFilter) (*domain.UserList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func setupAdminUserTestRouter(repo domain.UserRepository, jwtService *token.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	adminUserUsecase := user.NewAdminUserUsecase(repo, logger)
	adminUserHandler := handler.NewAdminUserHandler(adminUserUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/users", adminUserHandler.ListUsers)

	return router
}

func TestAdminUserHandler_ListUsers(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	t.Run("authorized admin successfully retrieves users", func(t *testing.T) {
		uid := uuid.New()
		now := time.Now()
		repo := &mockUserRepoForHandlerTest{
			listResult: &domain.UserList{
				Users: []*domain.User{
					{
						ID:        uid,
						FullName:  "Test User",
						Email:     "test@example.com",
						Phone:     "+919876543210",
						Role:      "CUSTOMER",
						Status:    "ACTIVE",
						CreatedAt: now,
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
				RoleCounts: domain.UserRoleCounts{
					Total:     1,
					Customers: 1,
				},
			},
		}

		router := setupAdminUserTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total"])
		users := data["users"].([]interface{})
		require.Len(t, users, 1)
		firstUser := users[0].(map[string]interface{})
		assert.Equal(t, uid.String(), firstUser["id"])
		assert.Equal(t, "Test User", firstUser["full_name"])
		assert.Nil(t, firstUser["password_hash"])
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		repo := &mockUserRepoForHandlerTest{}
		router := setupAdminUserTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("customer role returns 403 forbidden", func(t *testing.T) {
		repo := &mockUserRepoForHandlerTest{}
		router := setupAdminUserTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid page parameter returns 400 validation error", func(t *testing.T) {
		repo := &mockUserRepoForHandlerTest{}
		router := setupAdminUserTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=0", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

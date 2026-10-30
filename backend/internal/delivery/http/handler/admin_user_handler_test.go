package handler_test

import (
	"bytes"
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
	listResult  *domain.UserList
	listErr     error
	findByIDMap map[uuid.UUID]*domain.User
	updatedUser *domain.User
}

func (m *mockUserRepoForHandlerTest) List(filter domain.UserListFilter) (*domain.UserList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockUserRepoForHandlerTest) FindByID(id uuid.UUID) (*domain.User, error) {
	if m.findByIDMap != nil {
		if u, exists := m.findByIDMap[id]; exists {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForHandlerTest) Update(u *domain.User) error {
	m.updatedUser = u
	return nil
}

type mockTokenRepoForHandlerTest struct {
	domain.RefreshTokenRepository
	revokedUserIDs []uuid.UUID
}

func (m *mockTokenRepoForHandlerTest) RevokeAllByUserID(userID uuid.UUID) error {
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	return nil
}

func setupAdminUserTestRouter(repo domain.UserRepository, tokenRepo domain.RefreshTokenRepository, jwtService *token.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	adminUserUsecase := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
	adminUserHandler := handler.NewAdminUserHandler(adminUserUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id/block", adminUserHandler.BlockUser)
	admin.PATCH("/users/:id/unblock", adminUserHandler.UnblockUser)

	return router
}

func TestAdminUserHandler(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminID := uuid.New()
	adminToken, err := jwtService.GenerateAccessToken(adminID, "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	t.Run("authorized admin successfully retrieves users list", func(t *testing.T) {
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

		router := setupAdminUserTestRouter(repo, &mockTokenRepoForHandlerTest{}, jwtService)
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

	t.Run("get user by id returns sanitized details", func(t *testing.T) {
		targetID := uuid.New()
		now := time.Now()
		repo := &mockUserRepoForHandlerTest{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:           targetID,
					FullName:     "Detailed User",
					Email:        "detail@example.com",
					Phone:        "+919876543210",
					PasswordHash: "should-not-be-in-response",
					Role:         "CUSTOMER",
					Status:       "ACTIVE",
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
		}

		router := setupAdminUserTestRouter(repo, &mockTokenRepoForHandlerTest{}, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users/"+targetID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, targetID.String(), data["id"])
		assert.Equal(t, "Detailed User", data["full_name"])
		assert.Nil(t, data["password_hash"])
	})

	t.Run("block user updates status and revokes tokens", func(t *testing.T) {
		targetID := uuid.New()
		tokenRepo := &mockTokenRepoForHandlerTest{}
		repo := &mockUserRepoForHandlerTest{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Suspicious User",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		router := setupAdminUserTestRouter(repo, tokenRepo, jwtService)
		body := bytes.NewBufferString(`{"reason":"Fraudulent chargebacks"}`)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+targetID.String()+"/block", body)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		assert.Equal(t, "user blocked successfully", resp["message"])
		assert.Contains(t, tokenRepo.revokedUserIDs, targetID)
	})

	t.Run("unblock user restores status to ACTIVE", func(t *testing.T) {
		targetID := uuid.New()
		tokenRepo := &mockTokenRepoForHandlerTest{}
		repo := &mockUserRepoForHandlerTest{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Suspicious User",
					Role:     "CUSTOMER",
					Status:   "BLOCKED",
				},
			},
		}

		router := setupAdminUserTestRouter(repo, tokenRepo, jwtService)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+targetID.String()+"/unblock", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		assert.Equal(t, "user unblocked successfully", resp["message"])
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		repo := &mockUserRepoForHandlerTest{}
		router := setupAdminUserTestRouter(repo, &mockTokenRepoForHandlerTest{}, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("customer role returns 403 forbidden", func(t *testing.T) {
		repo := &mockUserRepoForHandlerTest{}
		router := setupAdminUserTestRouter(repo, &mockTokenRepoForHandlerTest{}, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

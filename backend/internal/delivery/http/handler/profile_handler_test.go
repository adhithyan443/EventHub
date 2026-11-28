package handler_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepoForHandler struct {
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForHandler) Create(u *domain.User) error { return nil }
func (m *mockUserRepoForHandler) FindByID(id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	userCopy := *u
	return &userCopy, nil
}
func (m *mockUserRepoForHandler) FindByEmail(email string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepoForHandler) FindByPhone(phone string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepoForHandler) Update(u *domain.User) error {
	m.users[u.ID] = u
	return nil
}
func (m *mockUserRepoForHandler) List(f domain.UserListFilter) (*domain.UserList, error) {
	return nil, nil
}

type mockRefreshTokenRepoForHandler struct{}

func (m *mockRefreshTokenRepoForHandler) Create(t *domain.RefreshToken) error { return nil }
func (m *mockRefreshTokenRepoForHandler) FindByTokenHash(h string) (*domain.RefreshToken, error) {
	return nil, domain.ErrRefreshTokenNotFound
}
func (m *mockRefreshTokenRepoForHandler) Revoke(id uuid.UUID) error            { return nil }
func (m *mockRefreshTokenRepoForHandler) RevokeAllByUserID(id uuid.UUID) error { return nil }

type mockTxReposForHandler struct {
	domain.TransactionRepositories
	userRepo *mockUserRepoForHandler
	rfRepo   *mockRefreshTokenRepoForHandler
}

func (m *mockTxReposForHandler) UserRepository() domain.UserRepository {
	return m.userRepo
}
func (m *mockTxReposForHandler) RefreshTokenRepository() domain.RefreshTokenRepository {
	return m.rfRepo
}

type mockTxManagerForHandler struct {
	txRepos *mockTxReposForHandler
}

func (m *mockTxManagerForHandler) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

func setupProfileHandlerTestEngine(
	userRepo *mockUserRepoForHandler,
	jwtSvc *token.JWTService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(nil))

	txRepos := &mockTxReposForHandler{
		userRepo: userRepo,
		rfRepo:   &mockRefreshTokenRepoForHandler{},
	}
	txMgr := &mockTxManagerForHandler{txRepos: txRepos}
	profileUC := profile.NewProfileUsecase(
		userRepo,
		nil,
		nil,
		txMgr,
		nil,
		nil,
		"test-secret",
		nil,
	)
	h := handler.NewProfileHandler(profileUC, slog.Default())

	rateLimiter := middleware.NewRateLimiter(2, time.Minute, slog.Default())

	profileGroup := r.Group("/api/v1/profile")
	profileGroup.Use(
		middleware.Auth(jwtSvc),
		middleware.RequireActiveUser(userRepo),
	)
	profileGroup.PUT("/password", rateLimiter.Middleware(), h.UpdatePassword)

	return r
}

func TestProfileHandler_UpdatePassword_HTTP(t *testing.T) {
	jwtSvc := token.NewJWTService("a-very-long-test-secret-at-least-32-bytes")
	userRepo := &mockUserRepoForHandler{
		users: make(map[uuid.UUID]*domain.User),
	}

	userID := uuid.New()
	oldPass := "CorrectOldPass123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPass), bcrypt.DefaultCost)
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		FullName:     "Profile Test User",
		Email:        "profiletest@example.com",
		PasswordHash: string(hashed),
		Role:         "CUSTOMER",
		Status:       "ACTIVE",
	}

	engine := setupProfileHandlerTestEngine(userRepo, jwtSvc)

	// 1. Unauthenticated request rejected
	t.Run("RejectsUnauthenticated", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password": oldPass,
			"new_password":     "BrandNewPass456!",
		})
		req := httptest.NewRequest("PUT", "/api/v1/profile/password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got: %d", w.Code)
		}
	})

	// 2. Authenticated request succeeds
	t.Run("AllowsAuthenticatedUser", func(t *testing.T) {
		tokenStr, err := jwtSvc.GenerateAccessToken(userID, "CUSTOMER")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		body, _ := json.Marshal(map[string]string{
			"current_password": oldPass,
			"new_password":     "BrandNewPass456!",
		})
		req := httptest.NewRequest("PUT", "/api/v1/profile/password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got: %d, body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != true {
			t.Fatalf("expected success true, got: %v", resp)
		}
	})

	// 3. User ID is taken from token, not body
	t.Run("IgnoresBodyUserID", func(t *testing.T) {
		tokenStr, _ := jwtSvc.GenerateAccessToken(userID, "CUSTOMER")

		otherUserID := uuid.New()
		body, _ := json.Marshal(map[string]string{
			"user_id":          otherUserID.String(),
			"current_password": "BrandNewPass456!",
			"new_password":     "AnotherNewPass789!",
		})
		req := httptest.NewRequest("PUT", "/api/v1/profile/password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK because it updates token's user, got: %d", w.Code)
		}
	})

	// 4. Rate limiter triggers after exceeding threshold
	t.Run("RateLimiterTriggers", func(t *testing.T) {
		tokenStr, _ := jwtSvc.GenerateAccessToken(userID, "CUSTOMER")

		// 3rd request should hit limit (rateLimiter was configured with limit=2 per minute)
		body, _ := json.Marshal(map[string]string{
			"current_password": "AnotherNewPass789!",
			"new_password":     "FinalPass999!",
		})
		req := httptest.NewRequest("PUT", "/api/v1/profile/password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 Too Many Requests, got: %d", w.Code)
		}
	})
}

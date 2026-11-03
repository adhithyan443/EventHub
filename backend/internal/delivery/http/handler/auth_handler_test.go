package handler_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAuthHandler struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForAuthHandler) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

type mockRefreshTokenRepoForAuthHandler struct {
	domain.RefreshTokenRepository
	tokens map[string]*domain.RefreshToken
}

func (m *mockRefreshTokenRepoForAuthHandler) FindByTokenHash(hash string) (*domain.RefreshToken, error) {
	if t, exists := m.tokens[hash]; exists {
		return t, nil
	}
	return nil, domain.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepoForAuthHandler) Revoke(id uuid.UUID) error {
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshTokenRepoForAuthHandler) Create(token *domain.RefreshToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func hashTokenForHandlerTest(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func setupAuthHandlerTestRouter(
	userRepo *mockUserRepoForAuthHandler,
	tokenRepo *mockRefreshTokenRepoForAuthHandler,
	jwtService *token.JWTService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	authUsecase := auth.NewAuthUsecase(
		userRepo,
		nil,
		tokenRepo,
		nil,
		jwtService,
		nil,
		nil,
		nil,
		logger,
	)

	authHandler := handler.NewAuthHandler(authUsecase, "http://localhost:5173")

	api := router.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.POST("/refresh-token", authHandler.RefreshToken)

	return router
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Jordan Lee",
		Email:     "jordan@example.com",
		Phone:     "+919876543210",
		Role:      "ORGANIZER",
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now(),
	}

	userRepo := &mockUserRepoForAuthHandler{
		users: map[uuid.UUID]*domain.User{
			userID: user,
		},
	}

	rawToken := "valid-refresh-token"
	tokenHash := hashTokenForHandlerTest(rawToken)

	tokenRepo := &mockRefreshTokenRepoForAuthHandler{
		tokens: map[string]*domain.RefreshToken{
			tokenHash: {
				ID:        uuid.New(),
				UserID:    userID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				RevokedAt: nil,
				CreatedAt: time.Now(),
			},
		},
	}

	router := setupAuthHandlerTestRouter(userRepo, tokenRepo, jwtService)

	bodyBytes, _ := json.Marshal(map[string]string{
		"refreshToken": rawToken,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh-token", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID       string `json:"id"`
			FullName string `json:"fullName"`
			Email    string `json:"email"`
			Phone    string `json:"phone"`
			Role     string `json:"role"`
			Status   string `json:"status"`
		} `json:"user"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, rawToken, resp.RefreshToken)

	// Verify authoritative user payload in response
	assert.Equal(t, userID.String(), resp.User.ID)
	assert.Equal(t, "Jordan Lee", resp.User.FullName)
	assert.Equal(t, "jordan@example.com", resp.User.Email)
	assert.Equal(t, "ORGANIZER", resp.User.Role)
	assert.Equal(t, "ACTIVE", resp.User.Status)

	// Validate claims inside the newly returned access token
	claims, err := jwtService.ValidateAccessToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims["userId"])
	assert.Equal(t, "ORGANIZER", claims["role"])
}

func TestAuthHandler_RefreshToken_MissingPayload(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	userRepo := &mockUserRepoForAuthHandler{}
	tokenRepo := &mockRefreshTokenRepoForAuthHandler{}

	router := setupAuthHandlerTestRouter(userRepo, tokenRepo, jwtService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh-token", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

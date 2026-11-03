package auth_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAuth struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForAuth) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

type mockRefreshTokenRepoForAuth struct {
	domain.RefreshTokenRepository
	tokens          map[string]*domain.RefreshToken
	revokedTokenIDs []uuid.UUID
	createdTokens   []*domain.RefreshToken
}

func (m *mockRefreshTokenRepoForAuth) FindByTokenHash(hash string) (*domain.RefreshToken, error) {
	if t, exists := m.tokens[hash]; exists {
		return t, nil
	}
	return nil, domain.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepoForAuth) Revoke(id uuid.UUID) error {
	m.revokedTokenIDs = append(m.revokedTokenIDs, id)
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshTokenRepoForAuth) Create(token *domain.RefreshToken) error {
	m.createdTokens = append(m.createdTokens, token)
	m.tokens[token.TokenHash] = token
	return nil
}

func hashTokenForTest(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func setupAuthUsecaseTest(
	userRepo *mockUserRepoForAuth,
	tokenRepo *mockRefreshTokenRepoForAuth,
	jwtService *token.JWTService,
) *auth.AuthUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return auth.NewAuthUsecase(
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
}

func TestRefreshToken_Success_ReturnsAuthoritativeUserAndNewRole(t *testing.T) {
	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	storedUser := &domain.User{
		ID:        userID,
		FullName:  "Alex Organizer",
		Email:     "alex@example.com",
		Phone:     "+919876543210",
		Role:      "ORGANIZER", // Role was promoted from CUSTOMER to ORGANIZER by admin approval
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	userRepo := &mockUserRepoForAuth{
		users: map[uuid.UUID]*domain.User{
			userID: storedUser,
		},
	}

	rawRefreshToken := "valid-refresh-token-string"
	tokenHash := hashTokenForTest(rawRefreshToken)
	storedTokenID := uuid.New()

	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			tokenHash: {
				ID:        storedTokenID,
				UserID:    userID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				RevokedAt: nil,
				CreatedAt: time.Now(),
			},
		},
	}

	uc := setupAuthUsecaseTest(userRepo, tokenRepo, jwtService)

	accessToken, newRefreshToken, returnedUser, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: rawRefreshToken,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, rawRefreshToken, newRefreshToken)

	// Verify authoritative user entity
	require.NotNil(t, returnedUser)
	assert.Equal(t, userID, returnedUser.ID)
	assert.Equal(t, "Alex Organizer", returnedUser.FullName)
	assert.Equal(t, "alex@example.com", returnedUser.Email)
	assert.Equal(t, "ORGANIZER", returnedUser.Role)
	assert.Equal(t, "ACTIVE", returnedUser.Status)

	// Verify access token claims contain the newly updated role
	claims, err := jwtService.ValidateAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims["userId"])
	assert.Equal(t, "ORGANIZER", claims["role"])

	// Verify rotation
	assert.Contains(t, tokenRepo.revokedTokenIDs, storedTokenID)
	assert.Len(t, tokenRepo.createdTokens, 1)
}

func TestRefreshToken_Validation_EmptyToken(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, &mockRefreshTokenRepoForAuth{}, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: "   ",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	}
}

func TestRefreshToken_TokenNotFound(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: "non-existent-token",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "invalid refresh token", err.Error())
}

func TestRefreshToken_TokenRevoked(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	raw := "revoked-token"
	revokedAt := time.Now().Add(-1 * time.Hour)
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    uuid.New(),
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: &revokedAt,
			},
		},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "invalid refresh token", err.Error())
}

func TestRefreshToken_TokenExpired(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	raw := "expired-token"
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    uuid.New(),
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(-10 * time.Minute),
				RevokedAt: nil,
			},
		},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "refresh token expired", err.Error())
}

func TestRefreshToken_UserInactive(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	userID := uuid.New()
	userRepo := &mockUserRepoForAuth{
		users: map[uuid.UUID]*domain.User{
			userID: {
				ID:     userID,
				Role:   "CUSTOMER",
				Status: "BLOCKED",
			},
		},
	}

	raw := "inactive-user-token"
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    userID,
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: nil,
			},
		},
	}

	uc := setupAuthUsecaseTest(userRepo, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "user account is not active", err.Error())
}

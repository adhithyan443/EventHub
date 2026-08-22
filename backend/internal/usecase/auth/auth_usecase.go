package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthUsecase struct {
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       *token.JWTService
	logger           *slog.Logger
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService *token.JWTService,
	logger *slog.Logger,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		logger:           logger,
	}
}

type RegisterInput struct {
	FullName string
	Email    string
	Password string
	Phone    string
}

type LoginInput struct {
	Email    string
	Password string
}

func generateRefreshToken() (string, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (u *AuthUsecase) Register(input RegisterInput) (*domain.User, error) {

	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)

	if input.FullName == "" {
		return nil, appErrors.NewValidationError("full name is required")
	}

	if input.Email == "" {
		return nil, appErrors.NewValidationError("email is required")
	}

	if input.Password == "" {
		return nil, appErrors.NewValidationError("password is required")
	}

	if input.Phone == "" {
		return nil, appErrors.NewValidationError("phone is required")
	}

	existingUser, err := u.userRepo.FindByEmail(input.Email)

	if err == nil && existingUser != nil {
		return nil, appErrors.NewConflictError("email already exists")
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	user := &domain.User{
		ID:           uuid.New(),
		FullName:     input.FullName,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(passwordHash),
		Role:         "Customer",
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUsecase) Login(input LoginInput) (*domain.User, string, string, error) {

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, "", "", appErrors.NewValidationError("email and password are required")
	}

	user, err := u.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, "", "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	if user.Status != "ACTIVE" {
		return nil, "", "", appErrors.NewUnauthorizedError("user account is not active")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	); err != nil {
		return nil, "", "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	accessToken, err := u.jwtService.GenerateAccessToken(
		user.ID,
		user.Role,
	)
	if err != nil {
		return nil, "", "", err
	}

	rawRefreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, "", "", err
	}

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashRefreshToken(rawRefreshToken),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := u.refreshTokenRepo.Create(refreshToken); err != nil {
		return nil, "", "", err
	}

	u.logger.Info(
		"user_login_success",
		"user_id", user.ID,
	)

	return user, accessToken, rawRefreshToken, nil

}

type RefreshTokenInput struct {
	RefreshToken string
}

func (u *AuthUsecase) RefreshToken(input RefreshTokenInput) (string, string, error) {
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)

	if input.RefreshToken == "" {
		return "", "", appErrors.NewValidationError("refresh token is required")
	}

	tokenHash := hashRefreshToken(input.RefreshToken)

	storedToken, err := u.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if storedToken.RevokedAt != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return "", "", appErrors.NewUnauthorizedError("refresh token expired")
	}

	user, err := u.userRepo.FindByID(storedToken.UserID)
	if err != nil {
		return "", "", appErrors.NewUnauthorizedError("invalid refresh token")
	}

	if user.Status != "ACTIVE" {
		return "", "", appErrors.NewUnauthorizedError("user account is not active")
	}

	accessToken, err := u.jwtService.GenerateAccessToken(
		user.ID,
		user.Role,
	)

	if err != nil {
		return "", "", err
	}

	if err := u.refreshTokenRepo.Revoke(storedToken.ID); err != nil {
		return "", "", err
	}
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	now := time.Now()

	newToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashRefreshToken(newRefreshToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := u.refreshTokenRepo.Create(newToken); err != nil {
		return "", "", err
	}

	u.logger.Info(
		"refresh_token_rotated",
		"user_id", user.ID,
	)

	return accessToken, newRefreshToken, nil
}

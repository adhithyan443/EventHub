package auth

import (
	"errors"
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
	userRepo   domain.UserRepository
	JWTService *token.JWTService
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	jwtService *token.JWTService,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:   userRepo,
		JWTService: jwtService,
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

func (u *AuthUsecase) Login(input LoginInput) (*domain.User, string, error) {

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, "", appErrors.NewValidationError("email and password are required")
	}

	user, err := u.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	if user.Status != "ACTIVE" {
		return nil, "", appErrors.NewUnauthorizedError("user account is not active")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	); err != nil {
		return nil, "", appErrors.NewUnauthorizedError("invalid email or password")
	}

	accessToken, err := u.JWTService.GenerateAccessToken(
		user.ID,
		user.Role,
	)
	if err != nil {
		return nil, "", err
	}

	return user, accessToken, nil

}

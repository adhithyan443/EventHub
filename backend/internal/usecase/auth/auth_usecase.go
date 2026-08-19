package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthUsecase struct {
	userRepo domain.UserRepository
}

func NewAuthUsecase(userRepo domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{
		userRepo: userRepo,
	}
}

type RegisterInput struct {
	FullName string
	Email    string
	Password string
	Phone    string
}

// type LoginInput struct{
// 	Email string
// 	Password string
// }

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

// func (u *AuthUsecase) Login(email, password string)(error){

// 	email = strings.ToLower(strings.TrimSpace(email));
// 	password = strings.TrimSpace(password)

// 	user,err := u.userRepo.FindByEmail(email);

// 	if err != nil {
// 		return appErrors.NewValidationError("invalid email or password")
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),[]byte(password))

// 	if err != nil{
// 		return appErrors.NewValidationError("invalid password")
// 	}

// }

package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID           uuid.UUID
	FullName     string
	Email        string
	Phone        string
	PasswordHash string
	Role         string
	Status       string
	ProfileImage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	Create(user *User) error
	FindByID(id uuid.UUID) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByPhone(phone string) (*User, error)
	Update(user *User) error
}

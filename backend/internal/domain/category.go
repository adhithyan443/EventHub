package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrCategoryNotFound = errors.New("category not found")

const (
	CategoryStatusActive   = "ACTIVE"
	CategoryStatusInactive = "INACTIVE"
)

type Category struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CategoryRepository interface {
	FindActive() ([]*Category, error)
}

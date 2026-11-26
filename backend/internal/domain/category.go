package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryNameDuplicate = errors.New("category name already exists")
	ErrCategoryInvalidName   = errors.New("invalid category name")
	ErrCategoryInvalidStatus = errors.New("invalid category status")
)

const (
	CategoryStatusActive   = "ACTIVE"
	CategoryStatusInactive = "INACTIVE"
)

type Category struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CategoryWithStats struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	EventCount  int64     `json:"event_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CategoryStatistics struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	InUse    int64 `json:"in_use"`
}

type CategoryListFilter struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
	Status string `json:"status"`
}

type CategoryListResult struct {
	Categories []*CategoryWithStats `json:"categories"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	Statistics CategoryStatistics   `json:"statistics"`
}

type CategoryRepository interface {
	FindActive() ([]*Category, error)
	FindByID(id uuid.UUID) (*Category, error)
	FindAll(filter CategoryListFilter) ([]*CategoryWithStats, int64, error)
	FindByName(name string) (*Category, error)
	Create(category *Category) error
	Update(category *Category) error
	UpdateStatus(id uuid.UUID, status string) error
	CountEventsByCategoryID(id uuid.UUID) (int64, error)
	GetStatistics() (*CategoryStatistics, error)
}

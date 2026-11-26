package category_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/google/uuid"
)

type mockCategoryRepo struct {
	categories []*domain.Category
	events     map[uuid.UUID]int64
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{
		categories: make([]*domain.Category, 0),
		events:     make(map[uuid.UUID]int64),
	}
}

func (m *mockCategoryRepo) FindActive() ([]*domain.Category, error) {
	active := make([]*domain.Category, 0)
	for _, c := range m.categories {
		if c.Status == domain.CategoryStatusActive {
			active = append(active, c)
		}
	}
	return active, nil
}

func (m *mockCategoryRepo) FindByID(id uuid.UUID) (*domain.Category, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (m *mockCategoryRepo) FindAll(filter domain.CategoryListFilter) ([]*domain.CategoryWithStats, int64, error) {
	var matched []*domain.CategoryWithStats
	for _, c := range m.categories {
		if filter.Search != "" {
			query := strings.ToLower(filter.Search)
			nameMatch := strings.Contains(strings.ToLower(c.Name), query)
			descMatch := strings.Contains(strings.ToLower(c.Description), query)
			if !nameMatch && !descMatch {
				continue
			}
		}
		if filter.Status != "" && strings.ToUpper(filter.Status) != "ALL" {
			if strings.ToUpper(c.Status) != strings.ToUpper(filter.Status) {
				continue
			}
		}
		eventCount := m.events[c.ID]
		matched = append(matched, &domain.CategoryWithStats{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Status:      c.Status,
			EventCount:  eventCount,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
	}
	total := int64(len(matched))
	return matched, total, nil
}

func (m *mockCategoryRepo) FindByName(name string) (*domain.Category, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, c := range m.categories {
		if strings.ToLower(strings.TrimSpace(c.Name)) == normalized {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (m *mockCategoryRepo) Create(c *domain.Category) error {
	m.categories = append(m.categories, c)
	return nil
}

func (m *mockCategoryRepo) Update(c *domain.Category) error {
	for i, existing := range m.categories {
		if existing.ID == c.ID {
			m.categories[i] = c
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (m *mockCategoryRepo) UpdateStatus(id uuid.UUID, status string) error {
	for _, c := range m.categories {
		if c.ID == id {
			c.Status = status
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (m *mockCategoryRepo) CountEventsByCategoryID(id uuid.UUID) (int64, error) {
	return m.events[id], nil
}

func (m *mockCategoryRepo) GetStatistics() (*domain.CategoryStatistics, error) {
	var total, active, inactive, inUse int64
	total = int64(len(m.categories))
	for _, c := range m.categories {
		if c.Status == domain.CategoryStatusActive {
			active++
		} else if c.Status == domain.CategoryStatusInactive {
			inactive++
		}
		if m.events[c.ID] > 0 {
			inUse++
		}
	}
	return &domain.CategoryStatistics{
		Total:    total,
		Active:   active,
		Inactive: inactive,
		InUse:    inUse,
	}, nil
}

func TestCategoryUsecase_GetActiveCategories(t *testing.T) {
	repo := newMockCategoryRepo()
	uc := category.NewCategoryUsecase(repo, slog.Default())

	cat1 := &domain.Category{ID: uuid.New(), Name: "Music", Status: domain.CategoryStatusActive}
	cat2 := &domain.Category{ID: uuid.New(), Name: "Comedy", Status: domain.CategoryStatusInactive}
	_ = repo.Create(cat1)
	_ = repo.Create(cat2)

	active, err := uc.GetActiveCategories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active category, got %d", len(active))
	}
	if active[0].Name != "Music" {
		t.Errorf("expected Music, got %s", active[0].Name)
	}
}

func TestCategoryUsecase_AdminListCategories(t *testing.T) {
	repo := newMockCategoryRepo()
	uc := category.NewCategoryUsecase(repo, slog.Default())

	cat1 := &domain.Category{ID: uuid.New(), Name: "Music Concerts", Status: domain.CategoryStatusActive}
	cat2 := &domain.Category{ID: uuid.New(), Name: "Standup Comedy", Status: domain.CategoryStatusInactive}
	cat3 := &domain.Category{ID: uuid.New(), Name: "Sports Football", Status: domain.CategoryStatusActive}
	_ = repo.Create(cat1)
	_ = repo.Create(cat2)
	_ = repo.Create(cat3)
	repo.events[cat1.ID] = 5

	ctx := context.Background()

	// List all
	res, err := uc.AdminListCategories(ctx, domain.CategoryListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("expected total 3, got %d", res.Total)
	}
	if res.Statistics.Active != 2 || res.Statistics.Inactive != 1 || res.Statistics.InUse != 1 {
		t.Errorf("unexpected stats: %+v", res.Statistics)
	}

	// Filter by status ACTIVE
	resActive, err := uc.AdminListCategories(ctx, domain.CategoryListFilter{Status: "ACTIVE"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resActive.Total != 2 {
		t.Errorf("expected 2 active, got %d", resActive.Total)
	}

	// Search filter
	resSearch, err := uc.AdminListCategories(ctx, domain.CategoryListFilter{Search: "comedy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resSearch.Total != 1 || resSearch.Categories[0].Name != "Standup Comedy" {
		t.Errorf("unexpected search result: %+v", resSearch)
	}
}

func TestCategoryUsecase_AdminCreateCategory(t *testing.T) {
	repo := newMockCategoryRepo()
	uc := category.NewCategoryUsecase(repo, slog.Default())
	ctx := context.Background()

	// Successful creation
	created, err := uc.AdminCreateCategory(ctx, "  Technology  ", "Tech conferences and meetups")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Name != "Technology" {
		t.Errorf("expected trimmed name 'Technology', got '%s'", created.Name)
	}
	if created.Status != domain.CategoryStatusActive {
		t.Errorf("expected new category to be ACTIVE, got %s", created.Status)
	}

	// Duplicate name - exact match
	_, err = uc.AdminCreateCategory(ctx, "Technology", "Duplicate")
	if !errors.Is(err, domain.ErrCategoryNameDuplicate) {
		t.Errorf("expected ErrCategoryNameDuplicate, got %v", err)
	}

	// Duplicate name - case-insensitive match
	_, err = uc.AdminCreateCategory(ctx, "technology", "Lower case duplicate")
	if !errors.Is(err, domain.ErrCategoryNameDuplicate) {
		t.Errorf("expected ErrCategoryNameDuplicate for case-insensitive duplicate, got %v", err)
	}

	// Invalid empty name
	_, err = uc.AdminCreateCategory(ctx, "   ", "Desc")
	if !errors.Is(err, domain.ErrCategoryInvalidName) {
		t.Errorf("expected ErrCategoryInvalidName for empty name, got %v", err)
	}

	// Invalid name length (< 2 chars)
	_, err = uc.AdminCreateCategory(ctx, "A", "Desc")
	if !errors.Is(err, domain.ErrCategoryInvalidName) {
		t.Errorf("expected ErrCategoryInvalidName for short name, got %v", err)
	}
}

func TestCategoryUsecase_AdminUpdateCategory(t *testing.T) {
	repo := newMockCategoryRepo()
	uc := category.NewCategoryUsecase(repo, slog.Default())
	ctx := context.Background()

	cat1, _ := uc.AdminCreateCategory(ctx, "Music", "Music events")
	_, _ = uc.AdminCreateCategory(ctx, "Sports", "Sports events")

	// Update cat1 successfully
	updated, err := uc.AdminUpdateCategory(ctx, cat1.ID, "Live Music", "Live performances")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Live Music" {
		t.Errorf("expected name 'Live Music', got '%s'", updated.Name)
	}

	// Update cat1 to cat2's name (duplicate collision)
	_, err = uc.AdminUpdateCategory(ctx, cat1.ID, "sports", "Collide with cat2")
	if !errors.Is(err, domain.ErrCategoryNameDuplicate) {
		t.Errorf("expected ErrCategoryNameDuplicate, got %v", err)
	}

	// Update cat1 with its same name (should succeed)
	_, err = uc.AdminUpdateCategory(ctx, cat1.ID, "Live Music", "Same name updated desc")
	if err != nil {
		t.Errorf("expected update with same name to succeed, got %v", err)
	}

	// Missing category ID
	_, err = uc.AdminUpdateCategory(ctx, uuid.New(), "New Name", "Desc")
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}

	// Empty name update
	_, err = uc.AdminUpdateCategory(ctx, cat1.ID, "", "Desc")
	if !errors.Is(err, domain.ErrCategoryInvalidName) {
		t.Errorf("expected ErrCategoryInvalidName, got %v", err)
	}
}

func TestCategoryUsecase_AdminToggleCategoryStatus(t *testing.T) {
	repo := newMockCategoryRepo()
	uc := category.NewCategoryUsecase(repo, slog.Default())
	ctx := context.Background()

	cat, _ := uc.AdminCreateCategory(ctx, "Festivals", "Festivals")
	if cat.Status != domain.CategoryStatusActive {
		t.Fatalf("expected ACTIVE")
	}

	// Deactivate
	deactivated, err := uc.AdminToggleCategoryStatus(ctx, cat.ID, domain.CategoryStatusInactive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deactivated.Status != domain.CategoryStatusInactive {
		t.Errorf("expected INACTIVE, got %s", deactivated.Status)
	}

	// Reactivate
	reactivated, err := uc.AdminToggleCategoryStatus(ctx, cat.ID, domain.CategoryStatusActive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reactivated.Status != domain.CategoryStatusActive {
		t.Errorf("expected ACTIVE, got %s", reactivated.Status)
	}

	// Invalid status
	_, err = uc.AdminToggleCategoryStatus(ctx, cat.ID, "DELETED")
	if !errors.Is(err, domain.ErrCategoryInvalidStatus) {
		t.Errorf("expected ErrCategoryInvalidStatus, got %v", err)
	}

	// Non-existent ID
	_, err = uc.AdminToggleCategoryStatus(ctx, uuid.New(), domain.CategoryStatusActive)
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}
}

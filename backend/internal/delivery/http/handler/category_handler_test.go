package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockCategoryRepoForHandler struct {
	categories []*domain.Category
	events     map[uuid.UUID]int64
}

func newMockCategoryRepoForHandler() *mockCategoryRepoForHandler {
	return &mockCategoryRepoForHandler{
		categories: make([]*domain.Category, 0),
		events:     make(map[uuid.UUID]int64),
	}
}

func (m *mockCategoryRepoForHandler) FindActive() ([]*domain.Category, error) {
	active := make([]*domain.Category, 0)
	for _, c := range m.categories {
		if c.Status == domain.CategoryStatusActive {
			active = append(active, c)
		}
	}
	return active, nil
}

func (m *mockCategoryRepoForHandler) FindByID(id uuid.UUID) (*domain.Category, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (m *mockCategoryRepoForHandler) FindAll(filter domain.CategoryListFilter) ([]*domain.CategoryWithStats, int64, error) {
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

func (m *mockCategoryRepoForHandler) FindByName(name string) (*domain.Category, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, c := range m.categories {
		if strings.ToLower(strings.TrimSpace(c.Name)) == normalized {
			return c, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (m *mockCategoryRepoForHandler) Create(c *domain.Category) error {
	m.categories = append(m.categories, c)
	return nil
}

func (m *mockCategoryRepoForHandler) Update(c *domain.Category) error {
	for i, existing := range m.categories {
		if existing.ID == c.ID {
			m.categories[i] = c
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (m *mockCategoryRepoForHandler) UpdateStatus(id uuid.UUID, status string) error {
	for _, c := range m.categories {
		if c.ID == id {
			c.Status = status
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (m *mockCategoryRepoForHandler) CountEventsByCategoryID(id uuid.UUID) (int64, error) {
	return m.events[id], nil
}

func (m *mockCategoryRepoForHandler) GetStatistics() (*domain.CategoryStatistics, error) {
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

type mockUserRepoForAuth struct {
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForAuth) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepoForAuth) Create(user *domain.User) error               { return nil }
func (m *mockUserRepoForAuth) FindByEmail(email string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepoForAuth) FindByPhone(phone string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepoForAuth) Update(user *domain.User) error                 { return nil }
func (m *mockUserRepoForAuth) List(filter domain.UserListFilter) (*domain.UserList, error) {
	return nil, nil
}

func setupCategoryTestRouter(repo *mockCategoryRepoForHandler, userRepo *mockUserRepoForAuth, jwtSvc *token.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(nil))

	uc := category.NewCategoryUsecase(repo, nil)
	h := handler.NewCategoryHandler(uc, nil)

	api := r.Group("/api/v1")

	// Public
	api.GET("/categories", h.GetCategories)

	// Admin protected
	admin := api.Group("/admin")
	if jwtSvc != nil && userRepo != nil {
		admin.Use(
			middleware.Auth(jwtSvc),
			middleware.RequireActiveUser(userRepo),
			middleware.RequireRole("ADMIN"),
		)
	}

	categories := admin.Group("/categories")
	{
		categories.GET("", h.AdminListCategories)
		categories.POST("", h.AdminCreateCategory)
		categories.PUT("/:id", h.AdminUpdateCategory)
		categories.PATCH("/:id/status", h.AdminToggleStatus)
	}

	return r
}

func TestCategoryHandler_PublicGetCategories_OnlyActive(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	r := setupCategoryTestRouter(repo, nil, nil)

	cat1 := &domain.Category{ID: uuid.New(), Name: "Active Category", Status: domain.CategoryStatusActive}
	cat2 := &domain.Category{ID: uuid.New(), Name: "Inactive Category", Status: domain.CategoryStatusInactive}
	_ = repo.Create(cat1)
	_ = repo.Create(cat2)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Categories []domain.Category `json:"categories"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success true")
	}
	if len(resp.Data.Categories) != 1 {
		t.Fatalf("expected only 1 category, got %d", len(resp.Data.Categories))
	}
	if resp.Data.Categories[0].Name != "Active Category" {
		t.Errorf("expected 'Active Category', got '%s'", resp.Data.Categories[0].Name)
	}
}

func TestCategoryHandler_AdminListCategories(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	r := setupCategoryTestRouter(repo, nil, nil)

	cat1 := &domain.Category{ID: uuid.New(), Name: "Concert", Status: domain.CategoryStatusActive}
	cat2 := &domain.Category{ID: uuid.New(), Name: "Workshop", Status: domain.CategoryStatusInactive}
	_ = repo.Create(cat1)
	_ = repo.Create(cat2)
	repo.events[cat1.ID] = 12

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/categories?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Categories []domain.CategoryWithStats `json:"categories"`
			Total      int64                      `json:"total"`
			Statistics domain.CategoryStatistics  `json:"statistics"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Total != 2 {
		t.Errorf("expected 2 total, got %d", resp.Data.Total)
	}
	if resp.Data.Statistics.InUse != 1 {
		t.Errorf("expected 1 in_use, got %d", resp.Data.Statistics.InUse)
	}
}

func TestCategoryHandler_AdminCreateCategory_SuccessAndValidation(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	r := setupCategoryTestRouter(repo, nil, nil)

	// Valid creation
	body, _ := json.Marshal(map[string]string{
		"name":        "Gaming & Esports",
		"description": "Tournaments and lan parties",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Duplicate creation (409 Conflict)
	dupBody, _ := json.Marshal(map[string]string{
		"name": "gaming & esports",
	})
	reqDup, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/categories", bytes.NewBuffer(dupBody))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate name, got %d: %s", wDup.Code, wDup.Body.String())
	}

	// Empty name (400 Validation Error)
	emptyBody, _ := json.Marshal(map[string]string{
		"name": "   ",
	})
	reqEmpty, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/categories", bytes.NewBuffer(emptyBody))
	reqEmpty.Header.Set("Content-Type", "application/json")
	wEmpty := httptest.NewRecorder()
	r.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty name, got %d", wEmpty.Code)
	}
}

func TestCategoryHandler_AdminUpdateCategory(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	r := setupCategoryTestRouter(repo, nil, nil)

	cat := &domain.Category{ID: uuid.New(), Name: "Original Name", Status: domain.CategoryStatusActive}
	other := &domain.Category{ID: uuid.New(), Name: "Other Name", Status: domain.CategoryStatusActive}
	_ = repo.Create(cat)
	_ = repo.Create(other)

	// Successful update
	updBody, _ := json.Marshal(map[string]string{
		"name":        "Updated Name",
		"description": "Updated description",
	})
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/categories/"+cat.ID.String(), bytes.NewBuffer(updBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Duplicate conflict with other category
	dupBody, _ := json.Marshal(map[string]string{
		"name": "other name",
	})
	reqDup, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/categories/"+cat.ID.String(), bytes.NewBuffer(dupBody))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", wDup.Code)
	}

	// Non-existent ID (404)
	req404, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/categories/"+uuid.New().String(), bytes.NewBuffer(updBody))
	req404.Header.Set("Content-Type", "application/json")
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)

	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", w404.Code)
	}
}

func TestCategoryHandler_AdminToggleStatus(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	r := setupCategoryTestRouter(repo, nil, nil)

	cat := &domain.Category{ID: uuid.New(), Name: "Test Category", Status: domain.CategoryStatusActive}
	_ = repo.Create(cat)

	// Deactivate
	body, _ := json.Marshal(map[string]string{"status": "INACTIVE"})
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/categories/"+cat.ID.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if cat.Status != domain.CategoryStatusInactive {
		t.Errorf("expected status to be INACTIVE, got %s", cat.Status)
	}

	// Reactivate
	reactBody, _ := json.Marshal(map[string]string{"status": "ACTIVE"})
	reqReact, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/categories/"+cat.ID.String()+"/status", bytes.NewBuffer(reactBody))
	reqReact.Header.Set("Content-Type", "application/json")
	wReact := httptest.NewRecorder()
	r.ServeHTTP(wReact, reqReact)

	if wReact.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wReact.Code, wReact.Body.String())
	}
	if cat.Status != domain.CategoryStatusActive {
		t.Errorf("expected status to be ACTIVE, got %s", cat.Status)
	}

	// Invalid status
	invBody, _ := json.Marshal(map[string]string{"status": "DELETED"})
	reqInv, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/categories/"+cat.ID.String()+"/status", bytes.NewBuffer(invBody))
	reqInv.Header.Set("Content-Type", "application/json")
	wInv := httptest.NewRecorder()
	r.ServeHTTP(wInv, reqInv)

	if wInv.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", wInv.Code)
	}
}

func TestCategoryHandler_RBAC_AccessControl(t *testing.T) {
	repo := newMockCategoryRepoForHandler()
	userRepo := &mockUserRepoForAuth{users: make(map[uuid.UUID]*domain.User)}
	jwtSvc := token.NewJWTService("secret-key-at-least-32-bytes-long!")

	adminID := uuid.New()
	customerID := uuid.New()

	userRepo.users[adminID] = &domain.User{
		ID:     adminID,
		Email:  "admin@eventhub.com",
		Role:   "ADMIN",
		Status: "ACTIVE",
	}
	userRepo.users[customerID] = &domain.User{
		ID:     customerID,
		Email:  "customer@eventhub.com",
		Role:   "CUSTOMER",
		Status: "ACTIVE",
	}

	adminToken, _ := jwtSvc.GenerateAccessToken(adminID, "ADMIN")
	customerToken, _ := jwtSvc.GenerateAccessToken(customerID, "CUSTOMER")

	r := setupCategoryTestRouter(repo, userRepo, jwtSvc)

	// 1. Unauthenticated request -> 401
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/categories", nil)
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", wUnauth.Code)
	}

	// 2. Customer user request -> 403 Forbidden
	reqCust, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/categories", nil)
	reqCust.Header.Set("Authorization", "Bearer "+customerToken)
	wCust := httptest.NewRecorder()
	r.ServeHTTP(wCust, reqCust)
	if wCust.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-admin, got %d", wCust.Code)
	}

	// 3. Admin user request -> 200 OK
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/categories", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)
	if wAdmin.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin, got %d: %s", wAdmin.Code, wAdmin.Body.String())
	}
}

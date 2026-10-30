package handler_test

import (
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
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerRepoForHandlerTest struct {
	domain.OrganizerRepository
	listResult *domain.AdminOrganizerList
	listErr    error
}

func (m *mockOrganizerRepoForHandlerTest) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func setupAdminOrganizerTestRouter(repo domain.OrganizerRepository, jwtService *token.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	adminOrganizerUsecase := organizer.NewAdminOrganizerUsecase(repo, logger)
	adminOrganizerHandler := handler.NewAdminOrganizerHandler(adminOrganizerUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/organizers", adminOrganizerHandler.ListOrganizers)

	return router
}

func TestAdminOrganizerHandler_ListOrganizers(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	require.NoError(t, err)

	customerToken, err := jwtService.GenerateAccessToken(uuid.New(), "CUSTOMER")
	require.NoError(t, err)

	t.Run("authorized admin successfully retrieves organizers", func(t *testing.T) {
		orgID := uuid.New()
		userID := uuid.New()
		now := time.Now()
		repo := &mockOrganizerRepoForHandlerTest{
			listResult: &domain.AdminOrganizerList{
				Organizers: []*domain.AdminOrganizerItem{
					{
						ID:                  orgID,
						UserID:              userID,
						UserName:            "John Doe",
						UserEmail:           "john@events.com",
						UserPhone:           "+919876543210",
						BusinessName:        "Global Events Ltd",
						BusinessType:        "Corporation",
						BusinessDescription: "Organizes international conferences",
						BusinessPhone:       "+911122334455",
						BusinessEmail:       "contact@globalevents.com",
						Website:             "https://globalevents.com",
						LogoURL:             "https://example.com/logo.jpg",
						GSTNumber:           "GST123456",
						PANNumber:           "PAN123456",
						Status:              "ACTIVE",
						CreatedAt:           now,
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}

		router := setupAdminOrganizerTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/organizers?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total"])
		organizers := data["organizers"].([]interface{})
		require.Len(t, organizers, 1)
		firstOrg := organizers[0].(map[string]interface{})
		assert.Equal(t, orgID.String(), firstOrg["id"])
		assert.Equal(t, "Global Events Ltd", firstOrg["business_name"])
		assert.Equal(t, "John Doe", firstOrg["user_name"])
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		repo := &mockOrganizerRepoForHandlerTest{}
		router := setupAdminOrganizerTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/organizers", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("customer role returns 403 forbidden", func(t *testing.T) {
		repo := &mockOrganizerRepoForHandlerTest{}
		router := setupAdminOrganizerTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/organizers", nil)
		req.Header.Set("Authorization", "Bearer "+customerToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid limit parameter returns 400 validation error", func(t *testing.T) {
		repo := &mockOrganizerRepoForHandlerTest{}
		router := setupAdminOrganizerTestRouter(repo, jwtService)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/organizers?limit=0", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

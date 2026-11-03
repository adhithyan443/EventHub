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

type mockAppRepoForMyApplicationHandler struct {
	domain.OrganizerApplicationRepository
	apps map[uuid.UUID]*domain.OrganizerApplication
}

func (m *mockAppRepoForMyApplicationHandler) FindByUserID(userID uuid.UUID) (*domain.OrganizerApplication, error) {
	if app, exists := m.apps[userID]; exists {
		return app, nil
	}
	return nil, domain.ErrOrganizerApplicationNotFound
}

type mockOrgRepoForMyApplicationHandler struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
}

func (m *mockOrgRepoForMyApplicationHandler) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if org, exists := m.organizers[userID]; exists {
		return org, nil
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockProfileRepoForMyApplicationHandler struct {
	domain.OrganizerProfileRepository
	profiles map[uuid.UUID]*domain.OrganizerProfile
}

func (m *mockProfileRepoForMyApplicationHandler) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerProfile, error) {
	if prof, exists := m.profiles[organizerID]; exists {
		return prof, nil
	}
	return nil, domain.ErrOrganizerProfileNotFound
}

func setupOrganizerApplicationTestRouter(
	appRepo domain.OrganizerApplicationRepository,
	orgRepo domain.OrganizerRepository,
	profRepo domain.OrganizerProfileRepository,
	jwtService *token.JWTService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	appUsecase := organizer.NewApplicationUsecase(appRepo, orgRepo, profRepo, nil, nil, nil, logger)
	appHandler := handler.NewOrganizerApplicationHandler(appUsecase, nil, logger)

	api := router.Group("/api/v1")
	organizerGroup := api.Group("/organizers")
	organizerGroup.Use(middleware.Auth(jwtService))
	organizerGroup.GET("/application", appHandler.GetMyApplication)

	return router
}

func TestGetMyApplication_ApprovedFallback_ReturnsApprovedStatus(t *testing.T) {
	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	orgID := uuid.New()

	userToken, err := jwtService.GenerateAccessToken(userID, "CUSTOMER")
	require.NoError(t, err)

	// Application is deleted from organizer_applications because admin approved it
	appRepo := &mockAppRepoForMyApplicationHandler{
		apps: map[uuid.UUID]*domain.OrganizerApplication{},
	}

	// Active organizer record exists
	orgRepo := &mockOrgRepoForMyApplicationHandler{
		organizers: map[uuid.UUID]*domain.Organizer{
			userID: {
				ID:        orgID,
				UserID:    userID,
				Status:    "ACTIVE",
				CreatedAt: time.Now().Add(-12 * time.Hour),
				UpdatedAt: time.Now(),
			},
		},
	}

	profRepo := &mockProfileRepoForMyApplicationHandler{
		profiles: map[uuid.UUID]*domain.OrganizerProfile{
			orgID: {
				OrganizerID:  orgID,
				BusinessName: "Global Stages Live",
				BusinessType: "COMPANY",
			},
		},
	}

	router := setupOrganizerApplicationTestRouter(appRepo, orgRepo, profRepo, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/application", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			BusinessName string `json:"business_name"`
			BusinessType string `json:"business_type"`
		} `json:"data"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, orgID.String(), resp.Data.ID)
	assert.Equal(t, "APPROVED", resp.Data.Status)
	assert.Equal(t, "Global Stages Live", resp.Data.BusinessName)
	assert.Equal(t, "COMPANY", resp.Data.BusinessType)
}

func TestGetMyApplication_MissingApplication_ReturnsNotFound(t *testing.T) {
	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	userToken, err := jwtService.GenerateAccessToken(userID, "CUSTOMER")
	require.NoError(t, err)

	appRepo := &mockAppRepoForMyApplicationHandler{
		apps: map[uuid.UUID]*domain.OrganizerApplication{},
	}
	orgRepo := &mockOrgRepoForMyApplicationHandler{
		organizers: map[uuid.UUID]*domain.Organizer{},
	}

	router := setupOrganizerApplicationTestRouter(appRepo, orgRepo, nil, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizers/application", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

package handler_test

import (
	"context"
	"encoding/json"
	"errors"
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
)

type stubOrganizerApplicationRepo struct {
	application *domain.OrganizerApplication
	err         error
}

func (s *stubOrganizerApplicationRepo) Create(application *domain.OrganizerApplication) error {
	return nil
}

func (s *stubOrganizerApplicationRepo) FindByUserID(userID uuid.UUID) (*domain.OrganizerApplication, error) {
	return nil, nil
}

func (s *stubOrganizerApplicationRepo) Update(application *domain.OrganizerApplication) error {
	return nil
}

func (s *stubOrganizerApplicationRepo) FindByID(id uuid.UUID) (*domain.OrganizerApplication, error) {
	return s.application, s.err
}

func (s *stubOrganizerApplicationRepo) Delete(id uuid.UUID) error {
	return nil
}

func (s *stubOrganizerApplicationRepo) List(page int, limit int, status domain.ApplicationStatus) (*domain.OrganizerApplicationList, error) {
	return nil, nil
}

type stubStoragePresigner struct {
	presignedURL string
	urlMap       map[string]string
	err          error
}

func (s *stubStoragePresigner) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.urlMap != nil {
		if u, exists := s.urlMap[key]; exists {
			return u, nil
		}
	}
	return s.presignedURL, nil
}

func setupAdminApplicationTestRouter(
	repo domain.OrganizerApplicationRepository,
	storage organizer.StoragePresigner,
	jwtService *token.JWTService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	adminUsecase := organizer.NewAdminApplicationUsecase(repo, nil, storage, logger)
	appHandler := handler.NewOrganizerApplicationHandler(nil, adminUsecase, logger)

	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)
	admin.GET("/organizer-applications/:id", appHandler.GetApplication)

	return router
}

func TestAdminGetApplication_AuthorizedWithDocument(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	appID := uuid.New()
	docKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	expectedURL := "https://test-bucket.s3.amazonaws.com/" + docKey + "?X-Amz-Signature=test1234"

	repo := &stubOrganizerApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  uuid.New(),
			ApplicantName:           "Alex Admin",
			ApplicantEmail:          "alex@example.com",
			ApplicantRole:           "CUSTOMER",
			BusinessName:            "Alex Productions",
			VerificationDocumentURL: docKey,
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &stubStoragePresigner{
		presignedURL: expectedURL,
	}

	router := setupAdminApplicationTestRouter(repo, storage, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+appID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                               `json:"success"`
		Data    organizer.AdminApplicationResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.Data.ID != appID {
		t.Errorf("expected application ID %s, got %s", appID, resp.Data.ID)
	}
	if resp.Data.ApplicantEmail != "alex@example.com" {
		t.Errorf("expected applicant email 'alex@example.com', got '%s'", resp.Data.ApplicantEmail)
	}
	if resp.Data.ApplicantRole != "CUSTOMER" {
		t.Errorf("expected applicant role 'CUSTOMER', got '%s'", resp.Data.ApplicantRole)
	}
	if resp.Data.VerificationDocumentURL != expectedURL {
		t.Errorf("expected verification document URL %s, got %s", expectedURL, resp.Data.VerificationDocumentURL)
	}
}

func TestAdminGetApplication_Unauthorized_MissingToken(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	router := setupAdminApplicationTestRouter(&stubOrganizerApplicationRepo{}, &stubStoragePresigner{}, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing token, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if resp["code"] != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got: %v", resp["code"])
	}
}

func TestAdminGetApplication_Unauthorized_NonAdminRole(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")

	roles := []string{"CUSTOMER", "ORGANIZER"}
	for _, role := range roles {
		t.Run("Role_"+role, func(t *testing.T) {
			userToken, err := jwtService.GenerateAccessToken(uuid.New(), role)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			router := setupAdminApplicationTestRouter(&stubOrganizerApplicationRepo{}, &stubStoragePresigner{}, jwtService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+uuid.New().String(), nil)
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 Forbidden for role %s, got %d", role, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json: %v", err)
			}
			if resp["code"] != "FORBIDDEN" {
				t.Errorf("expected code FORBIDDEN, got: %v", resp["code"])
			}
		})
	}
}

func TestAdminGetApplication_MissingDocument(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	appID := uuid.New()
	repo := &stubOrganizerApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  uuid.New(),
			ApplicantName:           "No Doc Applicant",
			BusinessName:            "No Doc Events",
			VerificationDocumentURL: "", // missing document
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &stubStoragePresigner{
		presignedURL: "https://should-never-be-used",
	}

	router := setupAdminApplicationTestRouter(repo, storage, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+appID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                              `json:"success"`
		Data    organizer.AdminApplicationResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp.Data.VerificationDocumentURL != "" {
		t.Errorf("expected empty verification document URL, got %s", resp.Data.VerificationDocumentURL)
	}
}

func TestAdminGetApplication_InvalidID(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	router := setupAdminApplicationTestRouter(&stubOrganizerApplicationRepo{}, &stubStoragePresigner{}, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/not-a-valid-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid UUID, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if resp["code"] != "VALIDATION_ERROR" || resp["message"] != "invalid application id" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAdminGetApplication_NonexistentApplication(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	appID := uuid.New()
	repo := &stubOrganizerApplicationRepo{
		err: domain.ErrOrganizerApplicationNotFound,
	}

	router := setupAdminApplicationTestRouter(repo, &stubStoragePresigner{}, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+appID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for nonexistent application, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if resp["code"] != "NOT_FOUND" || resp["message"] != "organizer application not found" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAdminGetApplication_PresignedURLGenerationFailure(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	appID := uuid.New()
	repo := &stubOrganizerApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  uuid.New(),
			ApplicantName:           "Alex Admin",
			BusinessName:            "Alex Productions",
			VerificationDocumentURL: "organizer-applications/" + appID.String() + "/verification-document.pdf",
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &stubStoragePresigner{
		err: errors.New("s3 connection failed"),
	}

	router := setupAdminApplicationTestRouter(repo, storage, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+appID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error on presign failure, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if resp["code"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected code INTERNAL_SERVER_ERROR, got: %+v", resp)
	}
}

func TestAdminGetApplication_AuthorizedWithLogoAndContactDetails(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	adminToken, err := jwtService.GenerateAccessToken(uuid.New(), "ADMIN")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	appID := uuid.New()
	logoKey := "organizer-applications/" + appID.String() + "/business-logo.png"
	docKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	expectedLogoURL := "https://test-bucket.s3.amazonaws.com/" + logoKey + "?signed=1"
	expectedDocURL := "https://test-bucket.s3.amazonaws.com/" + docKey + "?signed=2"

	repo := &stubOrganizerApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  uuid.New(),
			ApplicantName:           "Dev Admin",
			ApplicantEmail:          "dev@example.com",
			ApplicantRole:           "CUSTOMER",
			BusinessName:            "Dev Productions",
			LogoURL:                 logoKey,
			VerificationDocumentURL: docKey,
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &stubStoragePresigner{
		urlMap: map[string]string{
			logoKey: expectedLogoURL,
			docKey:  expectedDocURL,
		},
	}

	router := setupAdminApplicationTestRouter(repo, storage, jwtService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizer-applications/"+appID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                               `json:"success"`
		Data    organizer.AdminApplicationResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.Data.ApplicantEmail != "dev@example.com" {
		t.Errorf("expected applicant email 'dev@example.com', got '%s'", resp.Data.ApplicantEmail)
	}
	if resp.Data.ApplicantRole != "CUSTOMER" {
		t.Errorf("expected applicant role 'CUSTOMER', got '%s'", resp.Data.ApplicantRole)
	}
	if resp.Data.LogoURL != expectedLogoURL {
		t.Errorf("expected presigned logo URL %s, got %s", expectedLogoURL, resp.Data.LogoURL)
	}
	if resp.Data.VerificationDocumentURL != expectedDocURL {
		t.Errorf("expected presigned doc URL %s, got %s", expectedDocURL, resp.Data.VerificationDocumentURL)
	}
}

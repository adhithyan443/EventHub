package organizer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/google/uuid"
)

type mockApplicationRepo struct {
	application *domain.OrganizerApplication
	err         error
}

func (m *mockApplicationRepo) Create(application *domain.OrganizerApplication) error { return nil }
func (m *mockApplicationRepo) FindByUserID(userID uuid.UUID) (*domain.OrganizerApplication, error) {
	return nil, nil
}
func (m *mockApplicationRepo) Update(application *domain.OrganizerApplication) error { return nil }
func (m *mockApplicationRepo) FindByID(id uuid.UUID) (*domain.OrganizerApplication, error) {
	return m.application, m.err
}
func (m *mockApplicationRepo) Delete(id uuid.UUID) error { return nil }
func (m *mockApplicationRepo) List(page int, limit int, status domain.ApplicationStatus) (*domain.OrganizerApplicationList, error) {
	return nil, nil
}

type mockStoragePresigner struct {
	presignedURL  string
	urlMap        map[string]string
	err           error
	errMap        map[string]error
	calledWithKey string
	calledKeys    []string
	callCount     int
}

func (m *mockStoragePresigner) GetPresignedURL(ctx context.Context, key string) (string, error) {
	m.callCount++
	m.calledWithKey = key
	m.calledKeys = append(m.calledKeys, key)
	if m.errMap != nil {
		if err, exists := m.errMap[key]; exists {
			return "", err
		}
	}
	if m.err != nil {
		return "", m.err
	}
	if m.urlMap != nil {
		if u, exists := m.urlMap[key]; exists {
			return u, nil
		}
	}
	return m.presignedURL, nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetApplication_Success_GeneratesPresignedURL(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	documentKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	expectedPresignedURL := "https://test-bucket.s3.amazonaws.com/" + documentKey + "?X-Amz-Signature=dummy"

	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  userID,
			ApplicantName:           "Jane Doe",
			BusinessName:            "Acme Events",
			VerificationDocumentURL: documentKey,
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &mockStoragePresigner{
		presignedURL: expectedPresignedURL,
	}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ID != appID {
		t.Errorf("expected application ID %s, got %s", appID, result.ID)
	}
	if result.VerificationDocumentURL != expectedPresignedURL {
		t.Errorf("expected presigned URL %s, got %s", expectedPresignedURL, result.VerificationDocumentURL)
	}
	if storage.calledWithKey != documentKey {
		t.Errorf("expected storage to be called with key %s, got %s", documentKey, storage.calledWithKey)
	}
}

func TestGetApplication_MissingDocument_SkipsPresign(t *testing.T) {
	appID := uuid.New()
	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			VerificationDocumentURL: "",
			Status:                  domain.ApplicationPending,
		},
	}
	storage := &mockStoragePresigner{
		presignedURL: "https://should-not-be-called",
	}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.VerificationDocumentURL != "" {
		t.Errorf("expected empty verification document URL, got %s", result.VerificationDocumentURL)
	}
	if storage.callCount != 0 {
		t.Errorf("expected storage not to be called, got call count %d", storage.callCount)
	}
}

func TestGetApplication_NonExistentApplication_ReturnsNotFoundError(t *testing.T) {
	appID := uuid.New()
	repo := &mockApplicationRepo{
		err: domain.ErrOrganizerApplicationNotFound,
	}
	storage := &mockStoragePresigner{}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.HTTPStatus != 404 || appErr.Code != "NOT_FOUND" {
		t.Errorf("expected 404 NOT_FOUND, got status %d code %s", appErr.HTTPStatus, appErr.Code)
	}
}

func TestGetApplication_PresignFailure_ReturnsError(t *testing.T) {
	appID := uuid.New()
	documentKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			VerificationDocumentURL: documentKey,
			Status:                  domain.ApplicationPending,
		},
	}
	storage := &mockStoragePresigner{
		err: errors.New("s3 signature error"),
	}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if result != nil {
		t.Errorf("expected nil result on presign failure, got %+v", result)
	}
	if err == nil {
		t.Fatal("expected error on presign failure, got nil")
	}
	if err.Error() != "failed to generate verification document URL" {
		t.Errorf("expected 'failed to generate verification document URL', got: %v", err)
	}
}

func TestGetApplication_NilStorageService_ReturnsErrorWithoutPanic(t *testing.T) {
	appID := uuid.New()
	documentKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			VerificationDocumentURL: documentKey,
			Status:                  domain.ApplicationPending,
		},
	}

	// storageService is explicitly nil
	uc := organizer.NewAdminApplicationUsecase(repo, nil, nil, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if result != nil {
		t.Errorf("expected nil result when storageService is nil, got %+v", result)
	}
	if err == nil {
		t.Fatal("expected error when storageService is nil, got nil")
	}
	if err.Error() != "failed to generate verification document URL" {
		t.Errorf("expected 'failed to generate verification document URL', got: %v", err)
	}
}

func TestGetApplication_WithLogo_GeneratesPresignedLogoURL(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	logoKey := "organizer-applications/" + appID.String() + "/business-logo.png"
	docKey := "organizer-applications/" + appID.String() + "/verification-document.pdf"
	expectedLogoURL := "https://test-bucket.s3.amazonaws.com/" + logoKey + "?signed=true"
	expectedDocURL := "https://test-bucket.s3.amazonaws.com/" + docKey + "?signed=true"

	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			UserID:                  userID,
			ApplicantName:           "Jane Doe",
			ApplicantEmail:          "jane@example.com",
			ApplicantRole:           "CUSTOMER",
			BusinessName:            "Acme Events",
			LogoURL:                 logoKey,
			VerificationDocumentURL: docKey,
			Status:                  domain.ApplicationPending,
			CreatedAt:               time.Now(),
		},
	}
	storage := &mockStoragePresigner{
		urlMap: map[string]string{
			logoKey: expectedLogoURL,
			docKey:  expectedDocURL,
		},
	}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.LogoURL != expectedLogoURL {
		t.Errorf("expected presigned logo URL %s, got %s", expectedLogoURL, result.LogoURL)
	}
	if result.VerificationDocumentURL != expectedDocURL {
		t.Errorf("expected presigned doc URL %s, got %s", expectedDocURL, result.VerificationDocumentURL)
	}
	if storage.callCount != 2 {
		t.Errorf("expected storage to be called 2 times, got %d", storage.callCount)
	}
}

func TestGetApplication_PreservesContactEmailAndRole(t *testing.T) {
	appID := uuid.New()
	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:             appID,
			ApplicantName:  "Alice Smith",
			ApplicantEmail: "alice@example.com",
			ApplicantRole:  "CUSTOMER",
			Status:         domain.ApplicationPending,
		},
	}
	storage := &mockStoragePresigner{}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ApplicantEmail != "alice@example.com" {
		t.Errorf("expected applicant email 'alice@example.com', got '%s'", result.ApplicantEmail)
	}
	if result.ApplicantRole != "CUSTOMER" {
		t.Errorf("expected applicant role 'CUSTOMER', got '%s'", result.ApplicantRole)
	}
}

func TestGetApplication_LogoPresignFailure_ReturnsError(t *testing.T) {
	appID := uuid.New()
	logoKey := "organizer-applications/" + appID.String() + "/business-logo.png"
	repo := &mockApplicationRepo{
		application: &domain.OrganizerApplication{
			ID:                      appID,
			LogoURL:                 logoKey,
			VerificationDocumentURL: "",
			Status:                  domain.ApplicationPending,
		},
	}
	storage := &mockStoragePresigner{
		err: errors.New("s3 connection failed"),
	}

	uc := organizer.NewAdminApplicationUsecase(repo, nil, storage, newDiscardLogger())

	result, err := uc.GetApplication(context.Background(), appID)
	if result != nil {
		t.Errorf("expected nil result on logo presign failure, got %+v", result)
	}
	if err == nil {
		t.Fatal("expected error on logo presign failure, got nil")
	}
	if err.Error() != "failed to generate organizer logo URL" {
		t.Errorf("expected 'failed to generate organizer logo URL', got: %v", err)
	}
}

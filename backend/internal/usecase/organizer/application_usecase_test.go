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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAppRepoForGetApplication struct {
	domain.OrganizerApplicationRepository
	apps map[uuid.UUID]*domain.OrganizerApplication
}

func (m *mockAppRepoForGetApplication) FindByUserID(userID uuid.UUID) (*domain.OrganizerApplication, error) {
	if app, exists := m.apps[userID]; exists {
		return app, nil
	}
	return nil, domain.ErrOrganizerApplicationNotFound
}

type mockOrgRepoForGetApplication struct {
	domain.OrganizerRepository
	organizers map[uuid.UUID]*domain.Organizer
}

func (m *mockOrgRepoForGetApplication) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if org, exists := m.organizers[userID]; exists {
		return org, nil
	}
	return nil, domain.ErrOrganizerNotFound
}

type mockProfileRepoForGetApplication struct {
	domain.OrganizerProfileRepository
	profiles map[uuid.UUID]*domain.OrganizerProfile
}

func (m *mockProfileRepoForGetApplication) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerProfile, error) {
	if prof, exists := m.profiles[organizerID]; exists {
		return prof, nil
	}
	return nil, domain.ErrOrganizerProfileNotFound
}

type mockAddressRepoForGetApplication struct {
	domain.OrganizerAddressRepository
	addresses map[uuid.UUID]*domain.OrganizerAddress
}

func (m *mockAddressRepoForGetApplication) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerAddress, error) {
	if addr, exists := m.addresses[organizerID]; exists {
		return addr, nil
	}
	return nil, domain.ErrOrganizerAddressNotFound
}

func setupApplicationUsecaseTest(
	appRepo *mockAppRepoForGetApplication,
	orgRepo *mockOrgRepoForGetApplication,
	profRepo *mockProfileRepoForGetApplication,
	addrRepo *mockAddressRepoForGetApplication,
) *organizer.ApplicationUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return organizer.NewApplicationUsecase(
		appRepo,
		orgRepo,
		profRepo,
		addrRepo,
		nil,
		nil,
		logger,
	)
}

func TestGetApplication_ExistingPendingApplication(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	now := time.Now()

	app := &domain.OrganizerApplication{
		ID:          appID,
		UserID:      userID,
		BusinessName: "Summit Productions",
		BusinessType: "COMPANY",
		Description:  "Premier live event host",
		Phone:        "+919876543210",
		City:         "Mumbai",
		Status:       domain.ApplicationPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	appRepo := &mockAppRepoForGetApplication{
		apps: map[uuid.UUID]*domain.OrganizerApplication{
			userID: app,
		},
	}

	uc := setupApplicationUsecaseTest(appRepo, nil, nil, nil)

	res, err := uc.GetApplication(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, appID, res.ID)
	assert.Equal(t, "Summit Productions", res.BusinessName)
	assert.Equal(t, "COMPANY", res.BusinessType)
	assert.Equal(t, "Premier live event host", res.BusinessDescription)
	assert.Equal(t, "+919876543210", res.ContactPhone)
	assert.Equal(t, "Mumbai", res.City)
	assert.Equal(t, "PENDING", res.Status)
	assert.Empty(t, res.RejectionReason)
}

func TestGetApplication_ExistingRejectedApplication(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	now := time.Now()

	app := &domain.OrganizerApplication{
		ID:              appID,
		UserID:          userID,
		BusinessName:    "Apex Global",
		BusinessType:    "INDIVIDUAL",
		Description:     "Conference organizer",
		Phone:           "+919876543210",
		City:            "Chennai",
		Status:          domain.ApplicationRejected,
		RejectionReason: "Verification document illegible. Please re-upload PAN card.",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	appRepo := &mockAppRepoForGetApplication{
		apps: map[uuid.UUID]*domain.OrganizerApplication{
			userID: app,
		},
	}

	uc := setupApplicationUsecaseTest(appRepo, nil, nil, nil)

	res, err := uc.GetApplication(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, appID, res.ID)
	assert.Equal(t, "Apex Global", res.BusinessName)
	assert.Equal(t, "REJECTED", res.Status)
	assert.Equal(t, "Verification document illegible. Please re-upload PAN card.", res.RejectionReason)
}

func TestGetApplication_MissingApplication_WithActiveOrganizer(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	profID := uuid.New()
	addrID := uuid.New()
	now := time.Now().Add(-48 * time.Hour)

	// Application was deleted from organizer_applications upon admin approval
	appRepo := &mockAppRepoForGetApplication{
		apps: map[uuid.UUID]*domain.OrganizerApplication{},
	}

	// Active organizer record exists in organizers table
	orgRepo := &mockOrgRepoForGetApplication{
		organizers: map[uuid.UUID]*domain.Organizer{
			userID: {
				ID:        orgID,
				UserID:    userID,
				Status:    "ACTIVE",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	profRepo := &mockProfileRepoForGetApplication{
		profiles: map[uuid.UUID]*domain.OrganizerProfile{
			orgID: {
				ID:           profID,
				OrganizerID:  orgID,
				BusinessName: "Approved Global Events",
				BusinessType: "COMPANY",
				Description:  "Certified event management partner",
				Phone:        "+919876543210",
				Website:      "https://approvedevents.com",
				GSTNumber:    "29ABCDE1234F1Z5",
				PANNumber:    "ABCDE1234F",
			},
		},
	}

	addrRepo := &mockAddressRepoForGetApplication{
		addresses: map[uuid.UUID]*domain.OrganizerAddress{
			orgID: {
				ID:          addrID,
				OrganizerID: orgID,
				AddressLine: "123 Tech Park Road",
				City:        "Bengaluru",
				State:       "Karnataka",
				Country:     "India",
				PostalCode:  "560001",
			},
		},
	}

	uc := setupApplicationUsecaseTest(appRepo, orgRepo, profRepo, addrRepo)

	res, err := uc.GetApplication(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Fallback returns synthetic approved response
	assert.Equal(t, orgID, res.ID)
	assert.Equal(t, "APPROVED", res.Status)
	assert.Equal(t, "Approved Global Events", res.BusinessName)
	assert.Equal(t, "COMPANY", res.BusinessType)
	assert.Equal(t, "Certified event management partner", res.BusinessDescription)
	assert.Equal(t, "+919876543210", res.ContactPhone)
	assert.Equal(t, "https://approvedevents.com", res.Website)
	assert.Equal(t, "123 Tech Park Road", res.AddressLine)
	assert.Equal(t, "Bengaluru", res.City)
	assert.Equal(t, "Karnataka", res.State)
	assert.Equal(t, "India", res.Country)
	assert.Equal(t, "560001", res.PostalCode)
	assert.Equal(t, "29ABCDE1234F1Z5", res.GSTNumber)
	assert.Equal(t, "ABCDE1234F", res.PANNumber)
	assert.Empty(t, res.RejectionReason)
}

func TestGetApplication_MissingApplication_WithoutOrganizer(t *testing.T) {
	userID := uuid.New()

	appRepo := &mockAppRepoForGetApplication{
		apps: map[uuid.UUID]*domain.OrganizerApplication{},
	}

	orgRepo := &mockOrgRepoForGetApplication{
		organizers: map[uuid.UUID]*domain.Organizer{},
	}

	uc := setupApplicationUsecaseTest(appRepo, orgRepo, nil, nil)

	res, err := uc.GetApplication(context.Background(), userID)
	require.Error(t, err)
	assert.Nil(t, res)

	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, "NOT_FOUND", appErr.Code)
	assert.Equal(t, 404, appErr.HTTPStatus)
	assert.Equal(t, "organizer application not found", appErr.Message)
}

func TestGetApplication_OrganizerRecord_NonActiveStatus(t *testing.T) {
	nonActiveStatuses := []string{
		"SUSPENDED",
		"BLOCKED",
		"PENDING",
		"INACTIVE",
	}

	for _, status := range nonActiveStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			userID := uuid.New()
			orgID := uuid.New()

			appRepo := &mockAppRepoForGetApplication{
				apps: map[uuid.UUID]*domain.OrganizerApplication{},
			}

			orgRepo := &mockOrgRepoForGetApplication{
				organizers: map[uuid.UUID]*domain.Organizer{
					userID: {
						ID:     orgID,
						UserID: userID,
						Status: status,
					},
				},
			}

			uc := setupApplicationUsecaseTest(appRepo, orgRepo, nil, nil)

			res, err := uc.GetApplication(context.Background(), userID)
			require.Error(t, err)
			assert.Nil(t, res)

			var appErr *appErrors.AppError
			require.True(t, errors.As(err, &appErr))
			assert.Equal(t, "NOT_FOUND", appErr.Code)
			assert.Equal(t, 404, appErr.HTTPStatus)
			assert.Equal(t, "organizer application not found", appErr.Message)
		})
	}
}

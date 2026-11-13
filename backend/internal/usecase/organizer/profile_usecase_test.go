package organizer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerRepoForProfile struct {
	domain.OrganizerRepository
	organizer *domain.Organizer
	err       error
}

func (m *mockOrganizerRepoForProfile) FindByUserID(userID uuid.UUID) (*domain.Organizer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.organizer, nil
}

type mockProfileRepoForProfile struct {
	domain.OrganizerProfileRepository
	profile *domain.OrganizerProfile
	err     error
}

func (m *mockProfileRepoForProfile) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.profile, nil
}

type mockAddressRepoForProfile struct {
	domain.OrganizerAddressRepository
	address *domain.OrganizerAddress
	err     error
}

func (m *mockAddressRepoForProfile) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerAddress, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.address, nil
}

type mockBankAccountRepoForProfile struct {
	domain.OrganizerBankAccountRepository
	bankAccount *domain.OrganizerBankAccount
	err         error
}

func (m *mockBankAccountRepoForProfile) FindByOrganizerID(organizerID uuid.UUID) (*domain.OrganizerBankAccount, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.bankAccount, nil
}

func setupProfileUsecaseTest(
	org *domain.Organizer,
	prof *domain.OrganizerProfile,
	addr *domain.OrganizerAddress,
	bank *domain.OrganizerBankAccount,
	storage organizer.StoragePresigner,
) (*organizer.ProfileUsecase, *mockOrganizerRepoForProfile, *mockProfileRepoForProfile) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	orgRepo := &mockOrganizerRepoForProfile{organizer: org}
	profRepo := &mockProfileRepoForProfile{profile: prof}
	addrRepo := &mockAddressRepoForProfile{address: addr}
	bankRepo := &mockBankAccountRepoForProfile{bankAccount: bank}

	uc := organizer.NewProfileUsecase(
		orgRepo,
		profRepo,
		addrRepo,
		bankRepo,
		storage,
		logger,
	)

	return uc, orgRepo, profRepo
}

func TestProfileUsecase_GetProfile_LogoHandling(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	activeOrg := &domain.Organizer{
		ID:     orgID,
		UserID: userID,
		Status: "ACTIVE",
	}

	t.Run("resolves private S3 key to presigned URL", func(t *testing.T) {
		s3Key := "organizer-applications/3ec622d7-f661-4445-ae0a-75f75ea7935f/logo.png"
		expectedURL := "https://eventhub-assets-2026.s3.eu-north-1.amazonaws.com/" + s3Key + "?presigned=1"

		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      s3Key,
		}

		storage := &mockStoragePresigner{
			presignedURL: expectedURL,
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, storage)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, expectedURL, resp.LogoURL)
		assert.Equal(t, s3Key, storage.calledWithKey)
		assert.Equal(t, 1, storage.callCount)
	})

	t.Run("preserves external https URL verbatim without calling storage", func(t *testing.T) {
		externalURL := "https://cdn.example.com/images/organizer-logo.jpg"

		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      externalURL,
		}

		storage := &mockStoragePresigner{
			presignedURL: "https://unexpected-presigned-url",
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, storage)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, externalURL, resp.LogoURL)
		assert.Equal(t, 0, storage.callCount)
	})

	t.Run("preserves external http URL verbatim without calling storage", func(t *testing.T) {
		externalURL := "http://legacy.example.com/logo.png"

		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      externalURL,
		}

		storage := &mockStoragePresigner{
			presignedURL: "https://unexpected-presigned-url",
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, storage)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, externalURL, resp.LogoURL)
		assert.Equal(t, 0, storage.callCount)
	})

	t.Run("returns safe empty string for empty or whitespace logo key", func(t *testing.T) {
		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      "   ",
		}

		storage := &mockStoragePresigner{
			presignedURL: "https://unexpected-presigned-url",
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, storage)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.LogoURL)
		assert.Equal(t, 0, storage.callCount)
	})

	t.Run("presigning error logs warning and returns empty string without failing profile response", func(t *testing.T) {
		s3Key := "organizer-applications/invalid/logo.png"

		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      s3Key,
		}

		storage := &mockStoragePresigner{
			err: errors.New("s3 signature error or bucket unavailable"),
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, storage)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err, "profile retrieval must succeed even if logo presigning fails")
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.LogoURL)
		assert.Equal(t, "Acme Events", resp.BusinessName)
	})

	t.Run("returns empty logo when storageService is nil", func(t *testing.T) {
		s3Key := "organizer-applications/some/logo.png"

		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Acme Events",
			LogoURL:      s3Key,
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, nil, nil, nil)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.LogoURL)
	})
}

func TestProfileUsecase_GetProfile_AuthorizationAndErrors(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	t.Run("organizer not found returns forbidden error", func(t *testing.T) {
		uc, orgRepo, _ := setupProfileUsecaseTest(nil, nil, nil, nil, nil)
		orgRepo.err = domain.ErrOrganizerNotFound

		resp, err := uc.GetProfile(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
		assert.Contains(t, err.Error(), "organizer access required")
	})

	t.Run("organizer not active returns forbidden error", func(t *testing.T) {
		pendingOrg := &domain.Organizer{
			ID:     orgID,
			UserID: userID,
			Status: "PENDING",
		}

		uc, _, _ := setupProfileUsecaseTest(pendingOrg, nil, nil, nil, nil)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
		assert.Contains(t, err.Error(), "organizer account is not active")
	})

	t.Run("profile not found returns not found error", func(t *testing.T) {
		activeOrg := &domain.Organizer{
			ID:     orgID,
			UserID: userID,
			Status: "ACTIVE",
		}

		uc, _, profRepo := setupProfileUsecaseTest(activeOrg, nil, nil, nil, nil)
		profRepo.err = domain.ErrOrganizerProfileNotFound

		resp, err := uc.GetProfile(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
		assert.Contains(t, err.Error(), "organizer profile not found")
	})

	t.Run("maps address and bank account details correctly", func(t *testing.T) {
		activeOrg := &domain.Organizer{
			ID:     orgID,
			UserID: userID,
			Status: "ACTIVE",
		}
		prof := &domain.OrganizerProfile{
			OrganizerID:  orgID,
			BusinessName: "Grand Events",
			LogoURL:      "https://example.com/logo.png",
		}
		addr := &domain.OrganizerAddress{
			AddressLine: "123 Event Street",
			City:        "Bangalore",
			State:       "Karnataka",
			Country:     "India",
			PostalCode:  "560001",
		}
		bank := &domain.OrganizerBankAccount{
			BankName:               "HDFC Bank",
			AccountHolderName:      "Grand Events LLP",
			AccountNumberLast4:     "4321",
			IFSCCode:               "HDFC0001234",
		}

		uc, _, _ := setupProfileUsecaseTest(activeOrg, prof, addr, bank, nil)

		resp, err := uc.GetProfile(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "Grand Events", resp.BusinessName)
		assert.NotNil(t, resp.Address)
		assert.Equal(t, "123 Event Street", resp.Address.AddressLine)
		assert.Equal(t, "Bangalore", resp.Address.City)
		assert.NotNil(t, resp.BankAccount)
		assert.Equal(t, "HDFC Bank", resp.BankAccount.BankName)
		assert.Equal(t, "********4321", resp.BankAccount.AccountNumberMasked)
	})
}

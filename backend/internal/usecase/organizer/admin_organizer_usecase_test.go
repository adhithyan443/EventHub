package organizer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrganizerRepoForAdmin struct {
	domain.OrganizerRepository
	listResult *domain.AdminOrganizerList
	listErr    error
}

func (m *mockOrganizerRepoForAdmin) List(filter domain.AdminOrganizerListFilter) (*domain.AdminOrganizerList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func TestAdminOrganizerUsecase_ListOrganizers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful list returns mapped response with preserved external logo", func(t *testing.T) {
		orgID := uuid.New()
		userID := uuid.New()
		now := time.Now()

		repo := &mockOrganizerRepoForAdmin{
			listResult: &domain.AdminOrganizerList{
				Organizers: []*domain.AdminOrganizerItem{
					{
						ID:                  orgID,
						UserID:              userID,
						UserName:            "Bob Jones",
						UserEmail:           "bob@techevents.in",
						UserPhone:           "+919876543210",
						BusinessName:        "Tech Events India",
						BusinessType:        "Private Limited",
						BusinessDescription: "Organizing tech conferences.",
						BusinessPhone:       "+911122334455",
						BusinessEmail:       "info@techevents.in",
						Website:             "https://techevents.in",
						LogoURL:             "https://example.com/logo.png",
						GSTNumber:           "29ABCDE1234F1Z5",
						PANNumber:           "ABCDE1234F",
						Status:              "ACTIVE",
						CreatedAt:           now,
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}

		uc := organizer.NewAdminOrganizerUsecase(repo, nil, logger)
		resp, err := uc.ListOrganizers(context.Background(), 1, 10, "", "")

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(1), resp.Total)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.Limit)
		require.Len(t, resp.Organizers, 1)
		assert.Equal(t, orgID, resp.Organizers[0].ID)
		assert.Equal(t, userID, resp.Organizers[0].UserID)
		assert.Equal(t, "Tech Events India", resp.Organizers[0].BusinessName)
		assert.Equal(t, "Bob Jones", resp.Organizers[0].UserName)
		assert.Equal(t, "bob@techevents.in", resp.Organizers[0].UserEmail)
		assert.Equal(t, "ACTIVE", resp.Organizers[0].Status)
		assert.Equal(t, "https://example.com/logo.png", resp.Organizers[0].LogoURL)
	})

	t.Run("resolves private S3 logo key to presigned URL", func(t *testing.T) {
		orgID := uuid.New()
		s3Key := "organizer-applications/abc/logo.png"
		expectedPresigned := "https://s3.example.com/organizer-applications/abc/logo.png?presigned=true"

		repo := &mockOrganizerRepoForAdmin{
			listResult: &domain.AdminOrganizerList{
				Organizers: []*domain.AdminOrganizerItem{
					{
						ID:           orgID,
						BusinessName: "S3 Org",
						LogoURL:      s3Key,
						Status:       "ACTIVE",
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}

		storage := &mockStoragePresigner{
			presignedURL: expectedPresigned,
		}

		uc := organizer.NewAdminOrganizerUsecase(repo, storage, logger)
		resp, err := uc.ListOrganizers(context.Background(), 1, 10, "", "")

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Organizers, 1)
		assert.Equal(t, expectedPresigned, resp.Organizers[0].LogoURL)
		assert.Equal(t, s3Key, storage.calledWithKey)
	})

	t.Run("presigning error falls back to empty logo without failing list response", func(t *testing.T) {
		orgID := uuid.New()
		s3Key := "organizer-applications/broken/logo.png"

		repo := &mockOrganizerRepoForAdmin{
			listResult: &domain.AdminOrganizerList{
				Organizers: []*domain.AdminOrganizerItem{
					{
						ID:           orgID,
						BusinessName: "Broken Logo Org",
						LogoURL:      s3Key,
						Status:       "ACTIVE",
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}

		storage := &mockStoragePresigner{
			err: errors.New("s3 connection failed"),
		}

		uc := organizer.NewAdminOrganizerUsecase(repo, storage, logger)
		resp, err := uc.ListOrganizers(context.Background(), 1, 10, "", "")

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Organizers, 1)
		assert.Equal(t, "", resp.Organizers[0].LogoURL)
	})

	t.Run("empty logo key remains empty and does not call storage", func(t *testing.T) {
		orgID := uuid.New()

		repo := &mockOrganizerRepoForAdmin{
			listResult: &domain.AdminOrganizerList{
				Organizers: []*domain.AdminOrganizerItem{
					{
						ID:           orgID,
						BusinessName: "No Logo Org",
						LogoURL:      "",
						Status:       "ACTIVE",
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
			},
		}

		storage := &mockStoragePresigner{
			presignedURL: "https://unexpected.url",
		}

		uc := organizer.NewAdminOrganizerUsecase(repo, storage, logger)
		resp, err := uc.ListOrganizers(context.Background(), 1, 10, "", "")

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Organizers, 1)
		assert.Equal(t, "", resp.Organizers[0].LogoURL)
		assert.Equal(t, 0, storage.callCount)
	})

	t.Run("page less than 1 returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, nil, logger)
		_, err := uc.ListOrganizers(context.Background(), 0, 10, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page must be greater than zero")
	})

	t.Run("limit out of range returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, nil, logger)
		_, err := uc.ListOrganizers(context.Background(), 1, 150, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("invalid status filter returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, nil, logger)
		_, err := uc.ListOrganizers(context.Background(), 1, 10, "", "INVALID_STATUS")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("repository error returns generic error message", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{
			listErr: errors.New("db query timeout"),
		}
		uc := organizer.NewAdminOrganizerUsecase(repo, nil, logger)
		_, err := uc.ListOrganizers(context.Background(), 1, 10, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve organizers")
	})
}

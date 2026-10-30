package organizer_test

import (
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

	t.Run("successful list returns mapped response", func(t *testing.T) {
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

		uc := organizer.NewAdminOrganizerUsecase(repo, logger)
		resp, err := uc.ListOrganizers(1, 10, "", "")

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
	})

	t.Run("page less than 1 returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, logger)
		_, err := uc.ListOrganizers(0, 10, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page must be greater than zero")
	})

	t.Run("limit out of range returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, logger)
		_, err := uc.ListOrganizers(1, 150, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("invalid status filter returns error", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{}
		uc := organizer.NewAdminOrganizerUsecase(repo, logger)
		_, err := uc.ListOrganizers(1, 10, "", "INVALID_STATUS")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("repository error returns generic error message", func(t *testing.T) {
		repo := &mockOrganizerRepoForAdmin{
			listErr: errors.New("db query timeout"),
		}
		uc := organizer.NewAdminOrganizerUsecase(repo, logger)
		_, err := uc.ListOrganizers(1, 10, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve organizers")
	})
}

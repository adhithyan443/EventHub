package user_test

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAdmin struct {
	domain.UserRepository
	listResult *domain.UserList
	listErr    error
}

func (m *mockUserRepoForAdmin) List(filter domain.UserListFilter) (*domain.UserList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func TestAdminUserUsecase_ListUsers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful list returns mapped response and role counts", func(t *testing.T) {
		uid := uuid.New()
		now := time.Now()
		repo := &mockUserRepoForAdmin{
			listResult: &domain.UserList{
				Users: []*domain.User{
					{
						ID:           uid,
						FullName:     "Alice Smith",
						Email:        "alice@example.com",
						Phone:        "+1234567890",
						PasswordHash: "super-secret-hash-must-not-leak",
						Role:         "CUSTOMER",
						Status:       "ACTIVE",
						CreatedAt:    now,
					},
				},
				Total: 1,
				Page:  1,
				Limit: 10,
				RoleCounts: domain.UserRoleCounts{
					Total:     10,
					Customers: 8,
					Admins:    2,
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, logger)
		resp, err := uc.ListUsers(1, 10, "", "", "")

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(1), resp.Total)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.Limit)
		require.Len(t, resp.Users, 1)
		assert.Equal(t, uid, resp.Users[0].ID)
		assert.Equal(t, "Alice Smith", resp.Users[0].FullName)
		assert.Equal(t, "alice@example.com", resp.Users[0].Email)
		assert.Equal(t, "CUSTOMER", resp.Users[0].Role)
		assert.Equal(t, "ACTIVE", resp.Users[0].Status)
		assert.Equal(t, int64(10), resp.RoleCounts.Total)
		assert.Equal(t, int64(8), resp.RoleCounts.Customers)
	})

	t.Run("page less than 1 returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, logger)
		_, err := uc.ListUsers(0, 10, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page must be greater than zero")
	})

	t.Run("limit out of range returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, logger)
		_, err := uc.ListUsers(1, 101, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("invalid role filter returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, logger)
		_, err := uc.ListUsers(1, 10, "", "INVALID_ROLE", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role filter")
	})

	t.Run("invalid status filter returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, logger)
		_, err := uc.ListUsers(1, 10, "", "", "DELETED_OR_UNKNOWN")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("repository error returns generic error message", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{
			listErr: errors.New("db connection failure"),
		}
		uc := user.NewAdminUserUsecase(repo, logger)
		_, err := uc.ListUsers(1, 10, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve users")
	})
}

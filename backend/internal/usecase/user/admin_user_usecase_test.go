package user_test

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAdmin struct {
	domain.UserRepository
	listResult  *domain.UserList
	listErr     error
	findByIDMap map[uuid.UUID]*domain.User
	updateErr   error
	updatedUser *domain.User
}

func (m *mockUserRepoForAdmin) List(filter domain.UserListFilter) (*domain.UserList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockUserRepoForAdmin) FindByID(id uuid.UUID) (*domain.User, error) {
	if m.findByIDMap != nil {
		if u, exists := m.findByIDMap[id]; exists {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForAdmin) Update(u *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedUser = u
	return nil
}

type mockRefreshTokenRepoForAdmin struct {
	domain.RefreshTokenRepository
	revokedUserIDs []uuid.UUID
	revokeErr      error
}

func (m *mockRefreshTokenRepoForAdmin) RevokeAllByUserID(userID uuid.UUID) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	return nil
}

type mockTxRepositoriesForAdmin struct {
	domain.TransactionRepositories
	userRepo  domain.UserRepository
	tokenRepo domain.RefreshTokenRepository
}

func (m *mockTxRepositoriesForAdmin) UserRepository() domain.UserRepository {
	return m.userRepo
}

func (m *mockTxRepositoriesForAdmin) RefreshTokenRepository() domain.RefreshTokenRepository {
	return m.tokenRepo
}

type mockTxManagerForAdmin struct {
	txRepos    *mockTxRepositoriesForAdmin
	failTx     error
	committed  bool
	rolledBack bool
}

func (m *mockTxManagerForAdmin) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	if m.failTx != nil {
		m.rolledBack = true
		return m.failTx
	}
	err := fn(m.txRepos)
	if err != nil {
		m.rolledBack = true
		return err
	}
	m.committed = true
	return nil
}

func TestAdminUserUsecase_ListUsers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockTokenRepo := &mockRefreshTokenRepoForAdmin{}

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

		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
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

	t.Run("filter by status BLOCKED is valid", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{
			listResult: &domain.UserList{
				Users: []*domain.User{},
				Total: 0,
				Page:  1,
				Limit: 10,
			},
		}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		resp, err := uc.ListUsers(1, 10, "", "", "BLOCKED")
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("page less than 1 returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.ListUsers(0, 10, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page must be greater than zero")
	})

	t.Run("limit out of range returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.ListUsers(1, 101, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("invalid role filter returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.ListUsers(1, 10, "", "INVALID_ROLE", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role filter")
	})

	t.Run("invalid status filter returns error", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.ListUsers(1, 10, "", "", "DELETED_OR_UNKNOWN")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("repository error returns generic error message", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{
			listErr: errors.New("db connection failure"),
		}
		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.ListUsers(1, 10, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve users")
	})
}

func TestAdminUserUsecase_GetUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockTokenRepo := &mockRefreshTokenRepoForAdmin{}

	t.Run("successful get user returns details without password hash", func(t *testing.T) {
		uid := uuid.New()
		now := time.Now()
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				uid: {
					ID:           uid,
					FullName:     "John Doe",
					Email:        "john@example.com",
					Phone:        "+919876543210",
					PasswordHash: "super-secret-password-hash",
					Role:         "CUSTOMER",
					Status:       "ACTIVE",
					ProfileImage: "https://example.com/avatar.png",
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		res, err := uc.GetUser(uid)

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, uid, res.ID)
		assert.Equal(t, "John Doe", res.FullName)
		assert.Equal(t, "john@example.com", res.Email)
		assert.Equal(t, "+919876543210", res.Phone)
		assert.Equal(t, "CUSTOMER", res.Role)
		assert.Equal(t, "ACTIVE", res.Status)
		assert.Equal(t, "https://example.com/avatar.png", res.ProfileImage)
	})

	t.Run("user not found returns 404 AppError", func(t *testing.T) {
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{},
		}

		uc := user.NewAdminUserUsecase(repo, mockTokenRepo, nil, logger)
		_, err := uc.GetUser(uuid.New())

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}

func TestAdminUserUsecase_BlockUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminID := uuid.New()
	targetID := uuid.New()

	t.Run("successful block updates status and revokes tokens", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Bad Actor",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		res, err := uc.BlockUser(adminID, targetID, "Spamming events")

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, targetID, res.ID)
		assert.Equal(t, "BLOCKED", res.Status)
		require.NotNil(t, repo.updatedUser)
		assert.Equal(t, "BLOCKED", repo.updatedUser.Status)
		assert.Contains(t, tokenRepo.revokedUserIDs, targetID)
	})

	t.Run("self-blocking is rejected", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		_, err := uc.BlockUser(adminID, adminID, "Mistake")

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
		assert.Contains(t, appErr.Message, "cannot block own admin account")
	})

	t.Run("blocking another admin is rejected", func(t *testing.T) {
		fellowAdminID := uuid.New()
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				fellowAdminID: {
					ID:       fellowAdminID,
					FullName: "Fellow Admin",
					Role:     "ADMIN",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		_, err := uc.BlockUser(adminID, fellowAdminID, "Disagreement")

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
		assert.Contains(t, appErr.Message, "cannot block administrator account")
	})

	t.Run("already blocked user returns conflict", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Already Blocked",
					Role:     "CUSTOMER",
					Status:   "BLOCKED",
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		_, err := uc.BlockUser(adminID, targetID, "Again")

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Contains(t, appErr.Message, "user is already blocked")
	})

	t.Run("user not found returns 404", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		_, err := uc.BlockUser(adminID, uuid.New(), "Unknown")

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}

func TestAdminUserUsecase_BlockUser_Transaction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminID := uuid.New()
	targetID := uuid.New()

	t.Run("transaction commits both user update and token revocation atomically", func(t *testing.T) {
		txUserRepo := &mockUserRepoForAdmin{}
		txTokenRepo := &mockRefreshTokenRepoForAdmin{}
		txManager := &mockTxManagerForAdmin{
			txRepos: &mockTxRepositoriesForAdmin{
				userRepo:  txUserRepo,
				tokenRepo: txTokenRepo,
			},
		}

		mainUserRepo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Suspicious User",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(mainUserRepo, nil, txManager, logger)
		res, err := uc.BlockUser(adminID, targetID, "Fraud")

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, targetID, res.ID)
		assert.Equal(t, "BLOCKED", res.Status)
		assert.True(t, txManager.committed)
		assert.False(t, txManager.rolledBack)
		require.NotNil(t, txUserRepo.updatedUser)
		assert.Equal(t, "BLOCKED", txUserRepo.updatedUser.Status)
		assert.Contains(t, txTokenRepo.revokedUserIDs, targetID)
	})

	t.Run("transaction rolls back when user update fails", func(t *testing.T) {
		txUserRepo := &mockUserRepoForAdmin{
			updateErr: errors.New("db disk write failure"),
		}
		txTokenRepo := &mockRefreshTokenRepoForAdmin{}
		txManager := &mockTxManagerForAdmin{
			txRepos: &mockTxRepositoriesForAdmin{
				userRepo:  txUserRepo,
				tokenRepo: txTokenRepo,
			},
		}

		mainUserRepo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Suspicious User",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(mainUserRepo, nil, txManager, logger)
		res, err := uc.BlockUser(adminID, targetID, "Fraud")

		require.Error(t, err)
		assert.Nil(t, res)
		assert.True(t, txManager.rolledBack)
		assert.False(t, txManager.committed)
		assert.Empty(t, txTokenRepo.revokedUserIDs, "Tokens must not be revoked if user update fails")
	})

	t.Run("transaction rolls back when token revocation fails", func(t *testing.T) {
		txUserRepo := &mockUserRepoForAdmin{}
		txTokenRepo := &mockRefreshTokenRepoForAdmin{
			revokeErr: errors.New("token table locked"),
		}
		txManager := &mockTxManagerForAdmin{
			txRepos: &mockTxRepositoriesForAdmin{
				userRepo:  txUserRepo,
				tokenRepo: txTokenRepo,
			},
		}

		mainUserRepo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Suspicious User",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(mainUserRepo, nil, txManager, logger)
		res, err := uc.BlockUser(adminID, targetID, "Fraud")

		require.Error(t, err)
		assert.Nil(t, res)
		assert.True(t, txManager.rolledBack)
		assert.False(t, txManager.committed)
	})
}

func TestAdminUserUsecase_UnblockUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminID := uuid.New()
	targetID := uuid.New()

	t.Run("successful unblock restores status to ACTIVE", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Rehabilitated User",
					Role:     "CUSTOMER",
					Status:   "BLOCKED",
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		res, err := uc.UnblockUser(adminID, targetID)

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, targetID, res.ID)
		assert.Equal(t, "ACTIVE", res.Status)
		require.NotNil(t, repo.updatedUser)
		assert.Equal(t, "ACTIVE", repo.updatedUser.Status)
	})

	t.Run("already active user returns conflict", func(t *testing.T) {
		tokenRepo := &mockRefreshTokenRepoForAdmin{}
		repo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Active User",
					Role:     "CUSTOMER",
					Status:   "ACTIVE",
				},
			},
		}

		uc := user.NewAdminUserUsecase(repo, tokenRepo, nil, logger)
		_, err := uc.UnblockUser(adminID, targetID)

		require.Error(t, err)
		var appErr *appErrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Contains(t, appErr.Message, "user is already active")
	})
}

func TestAdminUserUsecase_UnblockUser_Transaction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminID := uuid.New()
	targetID := uuid.New()

	t.Run("transaction commits user unblock to ACTIVE", func(t *testing.T) {
		txUserRepo := &mockUserRepoForAdmin{}
		txManager := &mockTxManagerForAdmin{
			txRepos: &mockTxRepositoriesForAdmin{
				userRepo: txUserRepo,
			},
		}

		mainUserRepo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Rehabilitated User",
					Role:     "CUSTOMER",
					Status:   "BLOCKED",
				},
			},
		}

		uc := user.NewAdminUserUsecase(mainUserRepo, nil, txManager, logger)
		res, err := uc.UnblockUser(adminID, targetID)

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, targetID, res.ID)
		assert.Equal(t, "ACTIVE", res.Status)
		assert.True(t, txManager.committed)
		assert.False(t, txManager.rolledBack)
		require.NotNil(t, txUserRepo.updatedUser)
		assert.Equal(t, "ACTIVE", txUserRepo.updatedUser.Status)
	})

	t.Run("transaction rolls back when unblock update fails", func(t *testing.T) {
		txUserRepo := &mockUserRepoForAdmin{
			updateErr: errors.New("db error"),
		}
		txManager := &mockTxManagerForAdmin{
			txRepos: &mockTxRepositoriesForAdmin{
				userRepo: txUserRepo,
			},
		}

		mainUserRepo := &mockUserRepoForAdmin{
			findByIDMap: map[uuid.UUID]*domain.User{
				targetID: {
					ID:       targetID,
					FullName: "Rehabilitated User",
					Role:     "CUSTOMER",
					Status:   "BLOCKED",
				},
			},
		}

		uc := user.NewAdminUserUsecase(mainUserRepo, nil, txManager, logger)
		res, err := uc.UnblockUser(adminID, targetID)

		require.Error(t, err)
		assert.Nil(t, res)
		assert.True(t, txManager.rolledBack)
		assert.False(t, txManager.committed)
	})
}

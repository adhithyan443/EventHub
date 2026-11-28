package profile_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	users     map[uuid.UUID]*domain.User
	updateErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockUserRepo) Create(user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	// Return a copy
	userCopy := *u
	return &userCopy, nil
}

func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			userCopy := *u
			return &userCopy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByPhone(phone string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Phone == phone {
			userCopy := *u
			return &userCopy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Update(user *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) List(filter domain.UserListFilter) (*domain.UserList, error) {
	return &domain.UserList{}, nil
}

type mockRefreshTokenRepo struct {
	revokedUserIDs []uuid.UUID
	revokeErr      error
}

func (m *mockRefreshTokenRepo) Create(token *domain.RefreshToken) error {
	return nil
}

func (m *mockRefreshTokenRepo) FindByTokenHash(hash string) (*domain.RefreshToken, error) {
	return nil, domain.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepo) Revoke(id uuid.UUID) error {
	return nil
}

func (m *mockRefreshTokenRepo) RevokeAllByUserID(userID uuid.UUID) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	return nil
}

type mockTxRepos struct {
	domain.TransactionRepositories
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
}

func (m *mockTxRepos) UserRepository() domain.UserRepository {
	return m.userRepo
}

func (m *mockTxRepos) RefreshTokenRepository() domain.RefreshTokenRepository {
	return m.refreshTokenRepo
}

type mockTxManager struct {
	txRepos *mockTxRepos
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.txRepos)
}

func setupProfileUsecase(userRepo *mockUserRepo, refreshRepo *mockRefreshTokenRepo) *profile.ProfileUsecase {
	txRepos := &mockTxRepos{
		userRepo:         userRepo,
		refreshTokenRepo: refreshRepo,
	}
	txManager := &mockTxManager{txRepos: txRepos}

	return profile.NewProfileUsecase(
		userRepo,
		nil,
		nil,
		txManager,
		nil,
		nil,
		"test-secret",
		slog.Default(),
	)
}

func TestUpdatePassword_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	user := &domain.User{
		ID:           userID,
		FullName:     "Test User",
		Email:        "test@example.com",
		PasswordHash: string(hashed),
		Role:         "CUSTOMER",
		Status:       "ACTIVE",
		CreatedAt:    time.Now().Add(-24 * time.Hour),
		UpdatedAt:    time.Now().Add(-24 * time.Hour),
	}
	userRepo.users[userID] = user

	newPassword := "NewPassword456!"
	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify password hash updated
	updatedUser := userRepo.users[userID]
	if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(newPassword)); err != nil {
		t.Fatalf("expected new password to match stored hash, got error: %v", err)
	}

	// Verify refresh tokens were revoked
	if len(refreshRepo.revokedUserIDs) != 1 || refreshRepo.revokedUserIDs[0] != userID {
		t.Fatalf("expected refresh tokens to be revoked for user %s, got: %v", userID, refreshRepo.revokedUserIDs)
	}
}

func TestUpdatePassword_IncorrectCurrentPassword(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "CorrectPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: "WrongPassword999!",
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error for incorrect current password, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 401 || appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected 401 UNAUTHORIZED, got: %v", err)
	}
}

func TestUpdatePassword_WeakNewPassword(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	weakPasswords := []string{
		"short",           // too short
		"nouppercase123!", // no uppercase
		"NOLOWERCASE123!", // no lowercase
		"NoNumberSpecial", // no number
		"NoSpecial12345",  // no special char
	}

	for _, weak := range weakPasswords {
		err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
			CurrentPassword: oldPassword,
			NewPassword:     weak,
		})
		if err == nil {
			t.Fatalf("expected error for weak password %q, got nil", weak)
		}
		var appErr *appErrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
			t.Fatalf("expected VALIDATION_ERROR for weak password %q, got: %v", weak, err)
		}
	}
}

func TestUpdatePassword_Exceeds72Bytes(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	// 73 chars satisfying complexity
	over72Password := "A1!" + strings.Repeat("a", 70)

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: oldPassword,
		NewPassword:     over72Password,
	})

	if err == nil {
		t.Fatal("expected error for password exceeding 72 bytes, got nil")
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got: %v", err)
	}
}

func TestUpdatePassword_SamePassword(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	password := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: password,
		NewPassword:     password,
	})

	if err == nil {
		t.Fatal("expected error when new password is same as current password, got nil")
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got: %v", err)
	}
}

func TestUpdatePassword_GoogleOAuthUserWithoutPassword(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: "", // Google OAuth user
		Status:       "ACTIVE",
	}

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: "AnyPassword123!",
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error for user without password hash, got nil")
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got: %v", err)
	}
	if !strings.Contains(appErr.Message, "Google OAuth") {
		t.Fatalf("expected message to mention Google OAuth, got: %s", appErr.Message)
	}
}

func TestUpdatePassword_InactiveUser(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "BLOCKED",
	}

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: oldPassword,
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error for inactive user, got nil")
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 401 {
		t.Fatalf("expected 401 UNAUTHORIZED, got: %v", err)
	}
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	err := uc.UpdatePassword(context.Background(), uuid.New(), profile.UpdatePasswordInput{
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 404 {
		t.Fatalf("expected 404 NOT_FOUND, got: %v", err)
	}
}

func TestUpdatePassword_TransactionRollbackOnUpdateFail(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	// Force update failure in repo
	userRepo.updateErr = errors.New("db write failed")

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: oldPassword,
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error when db update fails, got nil")
	}

	// Verify token revocation was NOT executed / rolled back
	if len(refreshRepo.revokedUserIDs) != 0 {
		t.Fatalf("expected 0 token revocations on rollback, got: %d", len(refreshRepo.revokedUserIDs))
	}
}

func TestUpdatePassword_TransactionRollbackOnRevokeFail(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := &mockRefreshTokenRepo{}
	uc := setupProfileUsecase(userRepo, refreshRepo)

	oldPassword := "OldPassword123!"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	userID := uuid.New()
	userRepo.users[userID] = &domain.User{
		ID:           userID,
		PasswordHash: string(hashed),
		Status:       "ACTIVE",
	}

	// Force revoke failure
	refreshRepo.revokeErr = errors.New("token revoke db failed")

	err := uc.UpdatePassword(context.Background(), userID, profile.UpdatePasswordInput{
		CurrentPassword: oldPassword,
		NewPassword:     "NewPassword456!",
	})

	if err == nil {
		t.Fatal("expected error when token revocation fails, got nil")
	}
}

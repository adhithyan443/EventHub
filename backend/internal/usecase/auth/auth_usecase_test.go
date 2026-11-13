package auth_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAuth struct {
	domain.UserRepository
	users     map[uuid.UUID]*domain.User
	createErr error
}

func (m *mockUserRepoForAuth) FindByID(id uuid.UUID) (*domain.User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForAuth) FindByEmail(email string) (*domain.User, error) {
	norm := strings.ToLower(strings.TrimSpace(email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == norm {
			c := *u
			return &c, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForAuth) Create(u *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.users == nil {
		m.users = make(map[uuid.UUID]*domain.User)
	}
	m.users[u.ID] = u
	return nil
}

type mockRefreshTokenRepoForAuth struct {
	domain.RefreshTokenRepository
	tokens          map[string]*domain.RefreshToken
	revokedTokenIDs []uuid.UUID
	createdTokens   []*domain.RefreshToken
}

func (m *mockRefreshTokenRepoForAuth) FindByTokenHash(hash string) (*domain.RefreshToken, error) {
	if t, exists := m.tokens[hash]; exists {
		return t, nil
	}
	return nil, domain.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepoForAuth) Revoke(id uuid.UUID) error {
	m.revokedTokenIDs = append(m.revokedTokenIDs, id)
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshTokenRepoForAuth) Create(token *domain.RefreshToken) error {
	m.createdTokens = append(m.createdTokens, token)
	m.tokens[token.TokenHash] = token
	return nil
}

func hashTokenForTest(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func setupAuthUsecaseTest(
	userRepo *mockUserRepoForAuth,
	tokenRepo *mockRefreshTokenRepoForAuth,
	jwtService *token.JWTService,
) *auth.AuthUsecase {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return auth.NewAuthUsecase(
		userRepo,
		nil,
		tokenRepo,
		nil,
		jwtService,
		nil,
		nil,
		nil,
		logger,
	)
}

func TestRefreshToken_Success_ReturnsAuthoritativeUserAndNewRole(t *testing.T) {
	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	storedUser := &domain.User{
		ID:        userID,
		FullName:  "Alex Organizer",
		Email:     "alex@example.com",
		Phone:     "+919876543210",
		Role:      "ORGANIZER", // Role was promoted from CUSTOMER to ORGANIZER by admin approval
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	userRepo := &mockUserRepoForAuth{
		users: map[uuid.UUID]*domain.User{
			userID: storedUser,
		},
	}

	rawRefreshToken := "valid-refresh-token-string"
	tokenHash := hashTokenForTest(rawRefreshToken)
	storedTokenID := uuid.New()

	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			tokenHash: {
				ID:        storedTokenID,
				UserID:    userID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				RevokedAt: nil,
				CreatedAt: time.Now(),
			},
		},
	}

	uc := setupAuthUsecaseTest(userRepo, tokenRepo, jwtService)

	accessToken, newRefreshToken, returnedUser, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: rawRefreshToken,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, rawRefreshToken, newRefreshToken)

	// Verify authoritative user entity
	require.NotNil(t, returnedUser)
	assert.Equal(t, userID, returnedUser.ID)
	assert.Equal(t, "Alex Organizer", returnedUser.FullName)
	assert.Equal(t, "alex@example.com", returnedUser.Email)
	assert.Equal(t, "ORGANIZER", returnedUser.Role)
	assert.Equal(t, "ACTIVE", returnedUser.Status)

	// Verify access token claims contain the newly updated role
	claims, err := jwtService.ValidateAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims["userId"])
	assert.Equal(t, "ORGANIZER", claims["role"])

	// Verify rotation
	assert.Contains(t, tokenRepo.revokedTokenIDs, storedTokenID)
	assert.Len(t, tokenRepo.createdTokens, 1)
}

func TestRefreshToken_Validation_EmptyToken(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, &mockRefreshTokenRepoForAuth{}, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: "   ",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	}
}

func TestRefreshToken_TokenNotFound(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: "non-existent-token",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "invalid refresh token", err.Error())
}

func TestRefreshToken_TokenRevoked(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	raw := "revoked-token"
	revokedAt := time.Now().Add(-1 * time.Hour)
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    uuid.New(),
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: &revokedAt,
			},
		},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "invalid refresh token", err.Error())
}

func TestRefreshToken_TokenExpired(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	raw := "expired-token"
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    uuid.New(),
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(-10 * time.Minute),
				RevokedAt: nil,
			},
		},
	}
	uc := setupAuthUsecaseTest(&mockUserRepoForAuth{}, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "refresh token expired", err.Error())
}

func TestRefreshToken_UserInactive(t *testing.T) {
	jwtService := token.NewJWTService("01234567890123456789012345678901")
	userID := uuid.New()
	userRepo := &mockUserRepoForAuth{
		users: map[uuid.UUID]*domain.User{
			userID: {
				ID:     userID,
				Role:   "CUSTOMER",
				Status: "BLOCKED",
			},
		},
	}

	raw := "inactive-user-token"
	tokenRepo := &mockRefreshTokenRepoForAuth{
		tokens: map[string]*domain.RefreshToken{
			hashTokenForTest(raw): {
				ID:        uuid.New(),
				UserID:    userID,
				TokenHash: hashTokenForTest(raw),
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: nil,
			},
		},
	}

	uc := setupAuthUsecaseTest(userRepo, tokenRepo, jwtService)

	_, _, user, err := uc.RefreshToken(auth.RefreshTokenInput{
		RefreshToken: raw,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *appErrors.AppError
	if assert.True(t, errors.As(err, &appErr)) {
		assert.Equal(t, "UNAUTHORIZED", appErr.Code)
	}
	assert.Equal(t, "user account is not active", err.Error())
}

// --- VerifyOTP Tests with Advisory Lock & Email Reservation Integrity ---

type mockPendingRegRepoForAuth struct {
	domain.PendingRegistrationRepository
	registrations map[string]*domain.PendingRegistration
	updateErr     error
	deleteErr     error
}

func (m *mockPendingRegRepoForAuth) FindByEmail(email string) (*domain.PendingRegistration, error) {
	norm := strings.ToLower(strings.TrimSpace(email))
	if reg, ok := m.registrations[norm]; ok {
		c := *reg
		return &c, nil
	}
	return nil, domain.ErrPendingRegistrationNotFound
}

func (m *mockPendingRegRepoForAuth) Update(reg *domain.PendingRegistration) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	norm := strings.ToLower(strings.TrimSpace(reg.Email))
	c := *reg
	m.registrations[norm] = &c
	return nil
}

func (m *mockPendingRegRepoForAuth) Delete(id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for email, reg := range m.registrations {
		if reg.ID == id {
			delete(m.registrations, email)
			break
		}
	}
	return nil
}

type mockEmailChangeRepoForAuth struct {
	domain.EmailChangeRequestRepository
	requests   map[string]*domain.EmailChangeRequest
	deletedIDs []uuid.UUID
	deleteErr  error
	findErr    error
}

func (m *mockEmailChangeRepoForAuth) FindByNewEmail(email string) (*domain.EmailChangeRequest, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	norm := strings.ToLower(strings.TrimSpace(email))
	if req, ok := m.requests[norm]; ok {
		c := *req
		return &c, nil
	}
	return nil, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepoForAuth) DeleteByID(id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, id)
	for email, req := range m.requests {
		if req.ID == id {
			delete(m.requests, email)
			break
		}
	}
	return nil
}

type mockTxReposForAuth struct {
	domain.TransactionRepositories
	userRepo        domain.UserRepository
	pendingRegRepo  domain.PendingRegistrationRepository
	emailChangeRepo domain.EmailChangeRequestRepository
	lockErr         error
	lockedEmails    []string
}

func (r *mockTxReposForAuth) UserRepository() domain.UserRepository {
	return r.userRepo
}

func (r *mockTxReposForAuth) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegRepo
}

func (r *mockTxReposForAuth) EmailChangeRequestRepository() domain.EmailChangeRequestRepository {
	return r.emailChangeRepo
}

func (r *mockTxReposForAuth) AcquireEmailAdvisoryLock(email string) error {
	if r.lockErr != nil {
		return r.lockErr
	}
	r.lockedEmails = append(r.lockedEmails, email)
	return nil
}

type mockTxManagerForAuth struct {
	repos *mockTxReposForAuth
}

func (m *mockTxManagerForAuth) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

func setupVerifyOTPTest() (
	*auth.AuthUsecase,
	*mockUserRepoForAuth,
	*mockPendingRegRepoForAuth,
	*mockEmailChangeRepoForAuth,
	*mockTxReposForAuth,
) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userRepo := &mockUserRepoForAuth{users: make(map[uuid.UUID]*domain.User)}
	pendingRepo := &mockPendingRegRepoForAuth{registrations: make(map[string]*domain.PendingRegistration)}
	emailChangeRepo := &mockEmailChangeRepoForAuth{requests: make(map[string]*domain.EmailChangeRequest)}

	txRepos := &mockTxReposForAuth{
		userRepo:        userRepo,
		pendingRegRepo:  pendingRepo,
		emailChangeRepo: emailChangeRepo,
	}
	txManager := &mockTxManagerForAuth{repos: txRepos}

	uc := auth.NewAuthUsecase(
		userRepo,
		pendingRepo,
		nil,
		nil,
		nil,
		txManager,
		nil,
		nil,
		logger,
	)

	return uc, userRepo, pendingRepo, emailChangeRepo, txRepos
}

func hashAuthOTPForTest(otp string) string {
	h := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(h[:])
}

func TestVerifyOTP_Success(t *testing.T) {
	uc, userRepo, pendingRepo, _, txRepos := setupVerifyOTPTest()

	regID := uuid.New()
	email := "newuser@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           regID,
		FullName:     "New User",
		Email:        email,
		Phone:        "+919876543210",
		PasswordHash: "hashed-pw",
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
		OTPAttempts:  0,
	}

	createdUser, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.NoError(t, err)
	require.NotNil(t, createdUser)
	assert.Equal(t, "New User", createdUser.FullName)
	assert.Equal(t, email, createdUser.Email)
	assert.Equal(t, "CUSTOMER", createdUser.Role)
	assert.Equal(t, "ACTIVE", createdUser.Status)

	// Verify advisory lock was acquired
	assert.Contains(t, txRepos.lockedEmails, email)

	// Verify user persisted
	persisted, err := userRepo.FindByID(createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, email, persisted.Email)

	// Verify pending registration deleted
	_, findErr := pendingRepo.FindByEmail(email)
	assert.Equal(t, domain.ErrPendingRegistrationNotFound, findErr)
}

func TestVerifyOTP_ConcurrentEmailClaiming_UserAlreadyExists(t *testing.T) {
	uc, userRepo, pendingRepo, _, _ := setupVerifyOTPTest()

	regID := uuid.New()
	email := "concurrent@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           regID,
		FullName:     "Pending User",
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	// Concurrent flow already created user with this email
	existingID := uuid.New()
	userRepo.users[existingID] = &domain.User{
		ID:       existingID,
		FullName: "Existing User",
		Email:    email,
		Status:   "ACTIVE",
	}

	_, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, "CONFLICT", appErr.Code)
	assert.Equal(t, "email already exists", appErr.Message)

	// Pending registration must NOT be deleted
	p, findErr := pendingRepo.FindByEmail(email)
	require.NoError(t, findErr)
	assert.Equal(t, regID, p.ID)
}

func TestVerifyOTP_ActiveEmailChangeReservation_Conflict(t *testing.T) {
	uc, userRepo, pendingRepo, emailChangeRepo, _ := setupVerifyOTPTest()

	regID := uuid.New()
	email := "reserved@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           regID,
		FullName:     "Pending User",
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	// Another user has an active, unexpired email change reservation
	changeReqID := uuid.New()
	emailChangeRepo.requests[email] = &domain.EmailChangeRequest{
		ID:           changeReqID,
		UserID:       uuid.New(),
		NewEmail:     email,
		OTPExpiresAt: time.Now().Add(5 * time.Minute),
	}

	_, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, "CONFLICT", appErr.Code)
	assert.Equal(t, "email is reserved by another pending request", appErr.Message)

	// Pending registration preserved, user not created, change request not deleted
	assert.Len(t, userRepo.users, 0)
	assert.NotContains(t, emailChangeRepo.deletedIDs, changeReqID)
	_, findErr := pendingRepo.FindByEmail(email)
	require.NoError(t, findErr)
}

func TestVerifyOTP_ExpiredEmailChangeReservation_CleanedUp(t *testing.T) {
	uc, userRepo, pendingRepo, emailChangeRepo, txRepos := setupVerifyOTPTest()

	regID := uuid.New()
	email := "expiredreservation@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           regID,
		FullName:     "New User",
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	// Reservation has expired
	changeReqID := uuid.New()
	emailChangeRepo.requests[email] = &domain.EmailChangeRequest{
		ID:           changeReqID,
		UserID:       uuid.New(),
		NewEmail:     email,
		OTPExpiresAt: time.Now().Add(-5 * time.Minute),
	}

	createdUser, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.NoError(t, err)
	require.NotNil(t, createdUser)

	// Expired reservation was cleaned up
	assert.Contains(t, emailChangeRepo.deletedIDs, changeReqID)

	// User created and lock acquired
	assert.Contains(t, txRepos.lockedEmails, email)
	assert.Equal(t, 1, len(userRepo.users))
}

func TestVerifyOTP_RepositoryFailure_CreateUserError(t *testing.T) {
	uc, userRepo, pendingRepo, _, _ := setupVerifyOTPTest()

	email := "failcreate@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           uuid.New(),
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	userRepo.createErr = errors.New("db write timeout")

	_, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.Error(t, err)
	assert.Equal(t, "db write timeout", err.Error())

	// Pending registration must NOT be deleted if user create failed
	_, findErr := pendingRepo.FindByEmail(email)
	require.NoError(t, findErr)
}

func TestVerifyOTP_AdvisoryLockFailure_Propagated(t *testing.T) {
	uc, _, pendingRepo, _, txRepos := setupVerifyOTPTest()

	email := "lockfail@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           uuid.New(),
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	txRepos.lockErr = errors.New("postgres advisory lock deadlock")

	_, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.Error(t, err)
	assert.Equal(t, "postgres advisory lock deadlock", err.Error())
}

func TestVerifyOTP_ExpiredReservation_DeleteError_Propagated(t *testing.T) {
	uc, _, pendingRepo, emailChangeRepo, _ := setupVerifyOTPTest()

	email := "deletefail@example.com"
	rawOTP := "123456"

	pendingRepo.registrations[email] = &domain.PendingRegistration{
		ID:           uuid.New(),
		Email:        email,
		OTPHash:      hashAuthOTPForTest(rawOTP),
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	}

	emailChangeRepo.requests[email] = &domain.EmailChangeRequest{
		ID:           uuid.New(),
		NewEmail:     email,
		OTPExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	emailChangeRepo.deleteErr = errors.New("failed to delete expired reservation")

	_, err := uc.VerifyOTP(auth.VerifyOTPInput{
		Email: email,
		OTP:   rawOTP,
	})

	require.Error(t, err)
	assert.Equal(t, "failed to delete expired reservation", err.Error())
}

package profile_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockUserRepo struct {
	domain.UserRepository
	mu           sync.Mutex
	users        map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
	usersByPhone map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:        make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
		usersByPhone: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) AddUser(u *domain.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *u
	m.users[u.ID] = &copy
	m.usersByEmail[strings.ToLower(u.Email)] = &copy
	if u.Phone != "" {
		m.usersByPhone[u.Phone] = &copy
	}
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.usersByEmail[strings.ToLower(email)]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByPhone(phone string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.usersByPhone[phone]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Update(user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *user
	m.users[user.ID] = &copy
	m.usersByEmail[strings.ToLower(user.Email)] = &copy
	if user.Phone != "" {
		m.usersByPhone[user.Phone] = &copy
	}
	return nil
}

func (m *mockUserRepo) Create(user *domain.User) error {
	m.AddUser(user)
	return nil
}

type mockEmailChangeRepo struct {
	domain.EmailChangeRequestRepository
	mu       sync.Mutex
	requests map[uuid.UUID]*domain.EmailChangeRequest
}

func newMockEmailChangeRepo() *mockEmailChangeRepo {
	return &mockEmailChangeRepo{
		requests: make(map[uuid.UUID]*domain.EmailChangeRequest),
	}
}

func (m *mockEmailChangeRepo) Create(req *domain.EmailChangeRequest) error {
	return m.Upsert(req)
}

func (m *mockEmailChangeRepo) FindByUserID(userID uuid.UUID) (*domain.EmailChangeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req, ok := m.requests[userID]; ok {
		copy := *req
		return &copy, nil
	}
	return nil, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepo) FindByNewEmail(email string) (*domain.EmailChangeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	norm := strings.ToLower(email)
	for _, req := range m.requests {
		if strings.ToLower(req.NewEmail) == norm {
			copy := *req
			return &copy, nil
		}
	}
	return nil, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepo) Upsert(req *domain.EmailChangeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *req
	m.requests[req.UserID] = &copy
	return nil
}

func (m *mockEmailChangeRepo) IncrementAttempts(id uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, req := range m.requests {
		if req.ID == id {
			if req.OTPAttempts >= 5 {
				return req.OTPAttempts, nil
			}
			req.OTPAttempts++
			return req.OTPAttempts, nil
		}
	}
	return 0, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepo) UpdateResendCooldown(id uuid.UUID, userID uuid.UUID, availableAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.requests[userID]
	if !ok || req.ID != id {
		return domain.ErrEmailChangeRequestNotFound
	}
	req.ResendAvailableAt = availableAt
	return nil
}

func (m *mockEmailChangeRepo) RollbackResendCooldown(id uuid.UUID, userID uuid.UUID, currentCooldown time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.requests[userID]
	if !ok || req.ID != id {
		return nil
	}
	if req.ResendAvailableAt.Equal(currentCooldown) {
		req.ResendAvailableAt = time.Now().Add(-1 * time.Second)
	}
	return nil
}

func (m *mockEmailChangeRepo) UpdateOTPAndCooldown(id uuid.UUID, oldHash, newHash string, expiresAt, resendAt time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, req := range m.requests {
		if req.ID == id && req.OTPHash == oldHash {
			req.OTPHash = newHash
			req.OTPExpiresAt = expiresAt
			req.ResendAvailableAt = resendAt
			req.OTPAttempts = 0
			return true, nil
		}
	}
	return false, nil
}

func (m *mockEmailChangeRepo) DeleteByUserID(userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.requests, userID)
	return nil
}

func (m *mockEmailChangeRepo) DeleteByIDAndUserID(id uuid.UUID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req, ok := m.requests[userID]; ok && req.ID == id {
		delete(m.requests, userID)
	}
	return nil
}

func (m *mockEmailChangeRepo) DeleteByID(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for uid, req := range m.requests {
		if req.ID == id {
			delete(m.requests, uid)
			break
		}
	}
	return nil
}

type mockPendingRegistrationRepo struct {
	domain.PendingRegistrationRepository
	mu       sync.Mutex
	pendings map[string]*domain.PendingRegistration
}

func newMockPendingRegistrationRepo() *mockPendingRegistrationRepo {
	return &mockPendingRegistrationRepo{
		pendings: make(map[string]*domain.PendingRegistration),
	}
}

func (m *mockPendingRegistrationRepo) Create(pr *domain.PendingRegistration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *pr
	m.pendings[strings.ToLower(pr.Email)] = &copy
	return nil
}

func (m *mockPendingRegistrationRepo) FindByEmail(email string) (*domain.PendingRegistration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.pendings[strings.ToLower(email)]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, domain.ErrPendingRegistrationNotFound
}

func (m *mockPendingRegistrationRepo) Save(pr *domain.PendingRegistration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *pr
	m.pendings[strings.ToLower(pr.Email)] = &copy
	return nil
}

func (m *mockPendingRegistrationRepo) Delete(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for email, p := range m.pendings {
		if p.ID == id {
			delete(m.pendings, email)
			break
		}
	}
	return nil
}

type mockTxRepos struct {
	domain.TransactionRepositories
	userRepo                domain.UserRepository
	pendingRegistrationRepo domain.PendingRegistrationRepository
	emailChangeRepo         domain.EmailChangeRequestRepository
}

func (r *mockTxRepos) UserRepository() domain.UserRepository {
	return r.userRepo
}

func (r *mockTxRepos) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegistrationRepo
}

func (r *mockTxRepos) EmailChangeRequestRepository() domain.EmailChangeRequestRepository {
	return r.emailChangeRepo
}

func (r *mockTxRepos) AcquireEmailAdvisoryLock(email string) error {
	return nil
}

type mockTxManager struct {
	repos *mockTxRepos
	mu    sync.Mutex
}

func (m *mockTxManager) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(m.repos)
}

type mockEmailService struct {
	domain.EmailService
	mu           sync.Mutex
	sentOTPs     map[string][]string
	sendErr      error
	sendChangeFn func(to, otp string) error
}

func newMockEmailService() *mockEmailService {
	return &mockEmailService{
		sentOTPs: make(map[string][]string),
	}
}

func (m *mockEmailService) SendEmailChangeOTP(to, otp string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendChangeFn != nil {
		return m.sendChangeFn(to, otp)
	}
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentOTPs[to] = append(m.sentOTPs[to], otp)
	return nil
}

func (m *mockEmailService) LastOTP(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.sentOTPs[to]
	if len(list) == 0 {
		return ""
	}
	return list[len(list)-1]
}

type mockStorageService struct {
	mu          sync.Mutex
	uploaded    map[string][]byte
	deletedKeys []string
	uploadErr   error
	deleteErr   error
}

func newMockStorageService() *mockStorageService {
	return &mockStorageService{
		uploaded: make(map[string][]byte),
	}
}

func (m *mockStorageService) Upload(ctx context.Context, key string, file io.Reader, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.uploadErr != nil {
		return m.uploadErr
	}
	data, _ := io.ReadAll(file)
	m.uploaded[key] = data
	return nil
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedKeys = append(m.deletedKeys, key)
	delete(m.uploaded, key)
	return nil
}

func (m *mockStorageService) GetPresignedURL(ctx context.Context, key string) (string, error) {
	return "https://s3.example.com/" + key + "?presigned=true", nil
}

func setupProfileTest() (*profile.ProfileUsecase, *mockUserRepo, *mockEmailChangeRepo, *mockPendingRegistrationRepo, *mockEmailService, *mockStorageService) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userRepo := newMockUserRepo()
	emailChangeRepo := newMockEmailChangeRepo()
	pendingRegRepo := newMockPendingRegistrationRepo()
	emailService := newMockEmailService()
	storageService := newMockStorageService()

	txRepos := &mockTxRepos{
		userRepo:                userRepo,
		pendingRegistrationRepo: pendingRegRepo,
		emailChangeRepo:         emailChangeRepo,
	}
	txManager := &mockTxManager{repos: txRepos}

	uc := profile.NewProfileUsecase(
		userRepo,
		emailChangeRepo,
		pendingRegRepo,
		txManager,
		emailService,
		storageService,
		"test-jwt-secret-for-hmac-32chars",
		logger,
	)

	return uc, userRepo, emailChangeRepo, pendingRegRepo, emailService, storageService
}

func createTestPNGBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// --- Tests ---

func TestGetProfile(t *testing.T) {
	uc, userRepo, _, _, _, _ := setupProfileTest()
	ctx := context.Background()

	t.Run("success with s3 profile image resolves presigned URL", func(t *testing.T) {
		userID := uuid.New()
		user := &domain.User{
			ID:           userID,
			FullName:     "John Doe",
			Email:        "john@example.com",
			Phone:        "+919876543210",
			Role:         "CUSTOMER",
			ProfileImage: "profile-images/" + userID.String() + "/avatar.png",
			Status:       "ACTIVE",
		}
		userRepo.AddUser(user)

		resp, err := uc.GetProfile(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", resp.FullName)
		assert.Equal(t, "john@example.com", resp.Email)
		assert.True(t, strings.HasPrefix(resp.ProfileImage, "https://s3.example.com/profile-images/"))
	})

	t.Run("success with google avatar returns url directly", func(t *testing.T) {
		userID := uuid.New()
		googleURL := "https://lh3.googleusercontent.com/a/photo.jpg"
		user := &domain.User{
			ID:           userID,
			FullName:     "Jane Google",
			Email:        "jane@example.com",
			Role:         "ORGANIZER",
			ProfileImage: googleURL,
			Status:       "ACTIVE",
		}
		userRepo.AddUser(user)

		resp, err := uc.GetProfile(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, googleURL, resp.ProfileImage)
	})

	t.Run("user not found returns not found error", func(t *testing.T) {
		_, err := uc.GetProfile(ctx, uuid.New())
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}

func TestUpdateProfile(t *testing.T) {
	uc, userRepo, _, _, _, _ := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Original Name",
		Email:     "original@example.com",
		Phone:     "+919876543210",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	otherUserID := uuid.New()
	otherUser := &domain.User{
		ID:        otherUserID,
		FullName:  "Other User",
		Email:     "other@example.com",
		Phone:     "+919999999999",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(otherUser)

	t.Run("success update name and phone", func(t *testing.T) {
		resp, err := uc.UpdateProfile(ctx, userID, profile.UpdateProfileInput{
			FullName: "Updated Full Name",
			Phone:    "+919876543211",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Full Name", resp.FullName)
		assert.Equal(t, "+919876543211", resp.Phone)

		// Verify persisted
		persisted, err := userRepo.FindByID(userID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Full Name", persisted.FullName)
		assert.Equal(t, "+919876543211", persisted.Phone)
	})

	t.Run("validation failure: empty name", func(t *testing.T) {
		_, err := uc.UpdateProfile(ctx, userID, profile.UpdateProfileInput{
			FullName: "   ",
		})
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
	})

	t.Run("validation failure: invalid phone", func(t *testing.T) {
		_, err := uc.UpdateProfile(ctx, userID, profile.UpdateProfileInput{
			FullName: "Valid Name",
			Phone:    "12345",
		})
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
	})

	t.Run("conflict: phone already in use", func(t *testing.T) {
		_, err := uc.UpdateProfile(ctx, userID, profile.UpdateProfileInput{
			FullName: "Valid Name",
			Phone:    "+919999999999",
		})
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
	})
}

func TestRequestEmailChange(t *testing.T) {
	uc, userRepo, emailChangeRepo, pendingRegRepo, emailService, _ := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Test User",
		Email:     "current@example.com",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	t.Run("success email change request sends OTP", func(t *testing.T) {
		resp, err := uc.RequestEmailChange(ctx, userID, "NEW@example.com")
		require.NoError(t, err)
		assert.Equal(t, "new@example.com", resp.NewEmail)
		assert.Equal(t, 60, resp.ResendCooldownSeconds)
		assert.Equal(t, 600, resp.ExpiresInSeconds)

		// Verify OTP was sent to new email
		lastOTP := emailService.LastOTP("new@example.com")
		assert.Len(t, lastOTP, 6)

		// Verify saved in DB
		req, err := emailChangeRepo.FindByUserID(userID)
		require.NoError(t, err)
		assert.Equal(t, "new@example.com", req.NewEmail)
		assert.Equal(t, 0, req.OTPAttempts)
	})

	t.Run("reject same email as current", func(t *testing.T) {
		_, err := uc.RequestEmailChange(ctx, userID, "CURRENT@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
	})

	t.Run("reject invalid email format", func(t *testing.T) {
		_, err := uc.RequestEmailChange(ctx, userID, "not-an-email")
		assert.Error(t, err)
	})

	t.Run("conflict: email already registered in users", func(t *testing.T) {
		_ = emailChangeRepo.DeleteByUserID(userID)
		otherUser := &domain.User{
			ID:        uuid.New(),
			FullName:  "Other",
			Email:     "registered@example.com",
			Status:    "ACTIVE",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		userRepo.AddUser(otherUser)

		_, err := uc.RequestEmailChange(ctx, userID, "registered@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
	})

	t.Run("conflict: email reserved in active pending registration", func(t *testing.T) {
		_ = emailChangeRepo.DeleteByUserID(userID)
		_ = pendingRegRepo.Create(&domain.PendingRegistration{
			ID:           uuid.New(),
			Email:        "pending@example.com",
			OTPExpiresAt: time.Now().Add(5 * time.Minute),
		})

		_, err := uc.RequestEmailChange(ctx, userID, "pending@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
	})

	t.Run("conflict: email reserved in active change request by another user", func(t *testing.T) {
		_ = emailChangeRepo.DeleteByUserID(userID)
		otherID := uuid.New()
		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:           uuid.New(),
			UserID:       otherID,
			NewEmail:     "reserved@example.com",
			OTPExpiresAt: time.Now().Add(5 * time.Minute),
		})

		_, err := uc.RequestEmailChange(ctx, userID, "reserved@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.HTTPStatus)
	})

	t.Run("enforces resend cooldown on repeat request", func(t *testing.T) {
		// User already has an active request with resend cooldown in the future
		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:                uuid.New(),
			UserID:            userID,
			NewEmail:          "brandnew@example.com",
			ResendAvailableAt: time.Now().Add(45 * time.Second),
			OTPExpiresAt:      time.Now().Add(9 * time.Minute),
		})

		_, err := uc.RequestEmailChange(ctx, userID, "brandnew@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 429, appErr.HTTPStatus)
	})

	t.Run("smtp delivery failure cleans up request and returns error", func(t *testing.T) {
		emailService.sendErr = errors.New("smtp connection failed")
		defer func() { emailService.sendErr = nil }()

		// Clear existing request
		_ = emailChangeRepo.DeleteByUserID(userID)

		_, err := uc.RequestEmailChange(ctx, userID, "cleanuptest@example.com")
		assert.Error(t, err)

		// Request should have been cleaned up
		_, findErr := emailChangeRepo.FindByUserID(userID)
		assert.Equal(t, domain.ErrEmailChangeRequestNotFound, findErr)
	})
}

func TestResendEmailChangeOTP(t *testing.T) {
	uc, userRepo, emailChangeRepo, _, emailService, _ := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Test User",
		Email:     "old@example.com",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	t.Run("success resend resets cooldown", func(t *testing.T) {
		reqID := uuid.New()
		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:                reqID,
			UserID:            userID,
			NewEmail:          "resend@example.com",
			OTPHash:           "old-hash",
			ResendAvailableAt: time.Now().Add(-10 * time.Second), // expired cooldown
			OTPExpiresAt:      time.Now().Add(5 * time.Minute),
		})

		resp, err := uc.ResendEmailChangeOTP(ctx, userID, "resend@example.com")
		require.NoError(t, err)
		assert.Equal(t, "resend@example.com", resp.NewEmail)
		assert.Equal(t, 60, resp.ResendCooldownSeconds)

		// Verify new OTP was sent
		lastOTP := emailService.LastOTP("resend@example.com")
		assert.NotEmpty(t, lastOTP)
	})

	t.Run("cooldown still active returns 429", func(t *testing.T) {
		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:                uuid.New(),
			UserID:            userID,
			NewEmail:          "activecooldown@example.com",
			OTPHash:           "hash",
			ResendAvailableAt: time.Now().Add(30 * time.Second),
			OTPExpiresAt:      time.Now().Add(5 * time.Minute),
		})

		_, err := uc.ResendEmailChangeOTP(ctx, userID, "activecooldown@example.com")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 429, appErr.HTTPStatus)
	})

	t.Run("smtp failure rolls back cooldown", func(t *testing.T) {
		reqID := uuid.New()
		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:                reqID,
			UserID:            userID,
			NewEmail:          "smtpfail@example.com",
			OTPHash:           "old-hash",
			ResendAvailableAt: time.Now().Add(-10 * time.Second),
			OTPExpiresAt:      time.Now().Add(5 * time.Minute),
		})

		emailService.sendErr = errors.New("smtp down")
		defer func() { emailService.sendErr = nil }()

		_, err := uc.ResendEmailChangeOTP(ctx, userID, "smtpfail@example.com")
		assert.Error(t, err)

		// Verify cooldown was rolled back so user can retry immediately
		req, findErr := emailChangeRepo.FindByUserID(userID)
		require.NoError(t, findErr)
		assert.True(t, req.ResendAvailableAt.Before(time.Now()))
	})
}

func TestEmailChange_OTPAttempts_PersistAndBlockAtFifthAttempt(t *testing.T) {
	uc, userRepo, emailChangeRepo, _, _, _ := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Test User",
		Email:     "current@example.com",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	// Pre-create email change request with known HMAC
	mac := hmac.New(sha256.New, []byte("test-jwt-secret-for-hmac-32chars"))
	mac.Write([]byte("123456"))
	validHash := hex.EncodeToString(mac.Sum(nil))

	reqID := uuid.New()
	_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
		ID:           reqID,
		UserID:       userID,
		NewEmail:     "newemail@example.com",
		OTPHash:      validHash,
		OTPAttempts:  0,
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	})

	// Submit 4 incorrect attempts
	for i := 1; i <= 4; i++ {
		_, err := uc.VerifyEmailChange(ctx, userID, "newemail@example.com", "999999")
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
		assert.Contains(t, appErr.Message, "invalid verification code")

		// Verify attempt count was persisted in repository
		req, reqErr := emailChangeRepo.FindByUserID(userID)
		require.NoError(t, reqErr)
		assert.Equal(t, i, req.OTPAttempts)
	}

	// 5th incorrect attempt -> exceeds maximum attempts
	_, err := uc.VerifyEmailChange(ctx, userID, "newemail@example.com", "999999")
	assert.Error(t, err)
	var appErr *appErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.HTTPStatus)
	assert.Contains(t, strings.ToLower(appErr.Message), "maximum attempts reached")

	// Even if submitting the CORRECT OTP on the 6th attempt, it must be BLOCKED!
	_, err = uc.VerifyEmailChange(ctx, userID, "newemail@example.com", "123456")
	assert.Error(t, err)
	assert.True(t, errors.As(err, &appErr))
	assert.Contains(t, strings.ToLower(appErr.Message), "maximum verification attempts exceeded")

	// User's email must still be unchanged
	u, _ := userRepo.FindByID(userID)
	assert.Equal(t, "current@example.com", u.Email)
}

func TestVerifyEmailChange_SuccessAndOTPReplayProtection(t *testing.T) {
	uc, userRepo, emailChangeRepo, _, _, _ := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Test User",
		Email:     "before@example.com",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	mac := hmac.New(sha256.New, []byte("test-jwt-secret-for-hmac-32chars"))
	mac.Write([]byte("654321"))
	validHash := hex.EncodeToString(mac.Sum(nil))

	reqID := uuid.New()
	_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
		ID:           reqID,
		UserID:       userID,
		NewEmail:     "after@example.com",
		OTPHash:      validHash,
		OTPAttempts:  0,
		OTPExpiresAt: time.Now().Add(10 * time.Minute),
	})

	// Success verify
	resp, err := uc.VerifyEmailChange(ctx, userID, "after@example.com", "654321")
	require.NoError(t, err)
	assert.Equal(t, "after@example.com", resp.Email)

	// User model in DB updated
	updatedUser, err := userRepo.FindByID(userID)
	require.NoError(t, err)
	assert.Equal(t, "after@example.com", updatedUser.Email)
	assert.Equal(t, "ACTIVE", updatedUser.Status)

	// Request was burned/deleted from DB
	_, findErr := emailChangeRepo.FindByUserID(userID)
	assert.Equal(t, domain.ErrEmailChangeRequestNotFound, findErr)

	// Replay attempt fails
	_, replayErr := uc.VerifyEmailChange(ctx, userID, "after@example.com", "654321")
	assert.Error(t, replayErr)
}

func TestAvatarManagement(t *testing.T) {
	uc, userRepo, _, _, _, storageService := setupProfileTest()
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		FullName:  "Avatar User",
		Email:     "avatar@example.com",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.AddUser(user)

	pngBytes := createTestPNGBytes()

	t.Run("upload valid avatar updates user and resolves presigned url", func(t *testing.T) {
		resp, err := uc.UploadAvatar(ctx, userID, bytes.NewReader(pngBytes), "image/png", int64(len(pngBytes)))
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(resp.ProfileImage, "https://s3.example.com/profile-images/"))

		persisted, err := userRepo.FindByID(userID)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(persisted.ProfileImage, "profile-images/"))
		assert.True(t, strings.HasSuffix(persisted.ProfileImage, ".png"))
	})

	t.Run("upload new avatar cleans up old S3 image", func(t *testing.T) {
		persisted, _ := userRepo.FindByID(userID)
		oldKey := persisted.ProfileImage
		assert.NotEmpty(t, oldKey)

		// Upload second avatar
		_, err := uc.UploadAvatar(ctx, userID, bytes.NewReader(pngBytes), "image/png", int64(len(pngBytes)))
		require.NoError(t, err)

		// Verify oldKey was deleted from S3
		assert.Contains(t, storageService.deletedKeys, oldKey)
	})

	t.Run("upload new avatar preserves external google avatar", func(t *testing.T) {
		// Set user's avatar to external Google URL
		googleAvatar := "https://lh3.googleusercontent.com/a/google_avatar.jpg"
		googleUser := &domain.User{
			ID:           uuid.New(),
			FullName:     "Google User",
			Email:        "google@example.com",
			ProfileImage: googleAvatar,
			Status:       "ACTIVE",
		}
		userRepo.AddUser(googleUser)

		storageService.deletedKeys = nil

		_, err := uc.UploadAvatar(ctx, googleUser.ID, bytes.NewReader(pngBytes), "image/png", int64(len(pngBytes)))
		require.NoError(t, err)

		// Verify external Google URL was NOT deleted from S3
		assert.NotContains(t, storageService.deletedKeys, googleAvatar)
	})

	t.Run("rejects file larger than 5MB", func(t *testing.T) {
		_, err := uc.UploadAvatar(ctx, userID, bytes.NewReader(pngBytes), "image/png", 6*1024*1024)
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
	})

	t.Run("rejects unsupported MIME type", func(t *testing.T) {
		_, err := uc.UploadAvatar(ctx, userID, bytes.NewReader([]byte("plain text")), "text/plain", 10)
		assert.Error(t, err)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.HTTPStatus)
	})

	t.Run("remove avatar cleans up S3 image and clears user field", func(t *testing.T) {
		persisted, _ := userRepo.FindByID(userID)
		currentKey := persisted.ProfileImage
		assert.NotEmpty(t, currentKey)

		storageService.deletedKeys = nil

		resp, err := uc.RemoveAvatar(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, resp.ProfileImage)

		// User profileImage is cleared
		updated, _ := userRepo.FindByID(userID)
		assert.Empty(t, updated.ProfileImage)

		// S3 object deleted
		assert.Contains(t, storageService.deletedKeys, currentKey)
	})
}

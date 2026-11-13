package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Handler Test Mocks ---

type mockUserRepoForProfileHandler struct {
	domain.UserRepository
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForProfileHandler) FindByID(id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, exists := m.users[id]; exists {
		c := *u
		return &c, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForProfileHandler) FindByEmail(email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	norm := strings.ToLower(email)
	for _, u := range m.users {
		if strings.ToLower(u.Email) == norm {
			c := *u
			return &c, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForProfileHandler) FindByPhone(phone string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Phone == phone {
			c := *u
			return &c, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForProfileHandler) Update(u *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := *u
	m.users[u.ID] = &c
	return nil
}

type mockEmailChangeRepoForProfileHandler struct {
	domain.EmailChangeRequestRepository
	mu       sync.Mutex
	requests map[uuid.UUID]*domain.EmailChangeRequest
}

func (m *mockEmailChangeRepoForProfileHandler) FindByUserID(userID uuid.UUID) (*domain.EmailChangeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.requests[userID]; ok {
		c := *r
		return &c, nil
	}
	return nil, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepoForProfileHandler) FindByNewEmail(email string) (*domain.EmailChangeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	norm := strings.ToLower(email)
	for _, r := range m.requests {
		if strings.ToLower(r.NewEmail) == norm {
			c := *r
			return &c, nil
		}
	}
	return nil, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepoForProfileHandler) Upsert(req *domain.EmailChangeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := *req
	m.requests[req.UserID] = &c
	return nil
}

func (m *mockEmailChangeRepoForProfileHandler) IncrementAttempts(id uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.requests {
		if r.ID == id {
			if r.OTPAttempts >= 5 {
				return r.OTPAttempts, nil
			}
			r.OTPAttempts++
			return r.OTPAttempts, nil
		}
	}
	return 0, domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepoForProfileHandler) UpdateResendCooldown(id, userID uuid.UUID, availableAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.requests[userID]; ok && r.ID == id {
		r.ResendAvailableAt = availableAt
		return nil
	}
	return domain.ErrEmailChangeRequestNotFound
}

func (m *mockEmailChangeRepoForProfileHandler) RollbackResendCooldown(id, userID uuid.UUID, currentCooldown time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.requests[userID]; ok && r.ID == id && r.ResendAvailableAt.Equal(currentCooldown) {
		r.ResendAvailableAt = time.Now().Add(-1 * time.Second)
	}
	return nil
}

func (m *mockEmailChangeRepoForProfileHandler) UpdateOTPAndCooldown(id uuid.UUID, oldHash, newHash string, expiresAt, resendAt time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.requests {
		if r.ID == id && r.OTPHash == oldHash {
			r.OTPHash = newHash
			r.OTPExpiresAt = expiresAt
			r.ResendAvailableAt = resendAt
			r.OTPAttempts = 0
			return true, nil
		}
	}
	return false, nil
}

func (m *mockEmailChangeRepoForProfileHandler) DeleteByID(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for uid, r := range m.requests {
		if r.ID == id {
			delete(m.requests, uid)
			break
		}
	}
	return nil
}

func (m *mockEmailChangeRepoForProfileHandler) DeleteByIDAndUserID(id, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.requests[userID]; ok && r.ID == id {
		delete(m.requests, userID)
	}
	return nil
}

type mockPendingRegRepoForProfileHandler struct {
	domain.PendingRegistrationRepository
	mu sync.Mutex
}

func (m *mockPendingRegRepoForProfileHandler) FindByEmail(email string) (*domain.PendingRegistration, error) {
	return nil, domain.ErrPendingRegistrationNotFound
}

type mockTxReposForProfileHandler struct {
	domain.TransactionRepositories
	userRepo        domain.UserRepository
	emailChangeRepo domain.EmailChangeRequestRepository
	pendingRegRepo  domain.PendingRegistrationRepository
}

func (r *mockTxReposForProfileHandler) UserRepository() domain.UserRepository {
	return r.userRepo
}

func (r *mockTxReposForProfileHandler) EmailChangeRequestRepository() domain.EmailChangeRequestRepository {
	return r.emailChangeRepo
}

func (r *mockTxReposForProfileHandler) PendingRegistrationRepository() domain.PendingRegistrationRepository {
	return r.pendingRegRepo
}

func (r *mockTxReposForProfileHandler) AcquireEmailAdvisoryLock(email string) error {
	return nil
}

type mockTxManagerForProfileHandler struct {
	repos *mockTxReposForProfileHandler
}

func (m *mockTxManagerForProfileHandler) WithinTransaction(fn func(tx domain.TransactionRepositories) error) error {
	return fn(m.repos)
}

type mockEmailServiceForProfileHandler struct {
	domain.EmailService
}

func (m *mockEmailServiceForProfileHandler) SendEmailChangeOTP(to, otp string) error {
	return nil
}

type mockStorageServiceForProfileHandler struct {
	mu sync.Mutex
}

func (m *mockStorageServiceForProfileHandler) Upload(ctx context.Context, key string, file io.Reader, contentType string) error {
	return nil
}

func (m *mockStorageServiceForProfileHandler) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockStorageServiceForProfileHandler) GetPresignedURL(ctx context.Context, key string) (string, error) {
	return "https://s3.example.com/" + key + "?presigned=true", nil
}

func setupProfileHandlerTestRouter() (*gin.Engine, *mockUserRepoForProfileHandler, *mockEmailChangeRepoForProfileHandler, *token.JWTService, uuid.UUID, string) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(logger),
	)

	jwtSecret := "01234567890123456789012345678901"
	jwtService := token.NewJWTService(jwtSecret)

	userID := uuid.New()
	testUser := &domain.User{
		ID:        userID,
		FullName:  "Test User",
		Email:     "user@example.com",
		Phone:     "+919876543210",
		Role:      "CUSTOMER",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	userRepo := &mockUserRepoForProfileHandler{
		users: map[uuid.UUID]*domain.User{
			userID: testUser,
		},
	}

	emailChangeRepo := &mockEmailChangeRepoForProfileHandler{
		requests: make(map[uuid.UUID]*domain.EmailChangeRequest),
	}
	pendingRegRepo := &mockPendingRegRepoForProfileHandler{}

	txRepos := &mockTxReposForProfileHandler{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		pendingRegRepo:  pendingRegRepo,
	}
	txManager := &mockTxManagerForProfileHandler{repos: txRepos}
	emailService := &mockEmailServiceForProfileHandler{}
	storageService := &mockStorageServiceForProfileHandler{}

	uc := profile.NewProfileUsecase(
		userRepo,
		emailChangeRepo,
		pendingRegRepo,
		txManager,
		emailService,
		storageService,
		jwtSecret,
		logger,
	)

	h := handler.NewProfileHandler(uc, logger)

	api := router.Group("/api/v1")
	profGroup := api.Group("/profile")
	profGroup.Use(
		middleware.Auth(jwtService),
		middleware.RequireActiveUser(userRepo),
	)

	profGroup.GET("", h.GetProfile)
	profGroup.PUT("", h.UpdateProfile)
	profGroup.POST("/email/request-change", h.RequestEmailChange)
	profGroup.POST("/email/resend-otp", h.ResendEmailChangeOTP)
	profGroup.POST("/email/verify-change", h.VerifyEmailChange)
	profGroup.POST("/avatar", h.UploadAvatar)
	profGroup.DELETE("/avatar", h.RemoveAvatar)

	validToken, _ := jwtService.GenerateAccessToken(userID, "CUSTOMER")

	return router, userRepo, emailChangeRepo, jwtService, userID, validToken
}

func createTestPNGFile(t *testing.T) ([]byte, string) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	var imgBuf bytes.Buffer
	require.NoError(t, png.Encode(&imgBuf, img))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "test.png")
	require.NoError(t, err)
	_, err = part.Write(imgBuf.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return body.Bytes(), writer.FormDataContentType()
}

// --- Tests ---

func TestProfileHandler_GetProfile(t *testing.T) {
	router, _, _, _, _, validToken := setupProfileHandlerTestRouter()

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("authenticated request returns 200 with profile data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Test User", data["full_name"])
		assert.Equal(t, "user@example.com", data["email"])
	})
}

func TestProfileHandler_UpdateProfile(t *testing.T) {
	router, _, _, _, _, validToken := setupProfileHandlerTestRouter()

	t.Run("valid update returns 200", func(t *testing.T) {
		body := `{"full_name":"New Name","phone":"+919876543219"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "New Name", data["full_name"])
	})

	t.Run("invalid name returns 400", func(t *testing.T) {
		body := `{"full_name":"","phone":"+919876543219"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestProfileHandler_EmailChangeFlow(t *testing.T) {
	router, _, emailChangeRepo, _, userID, validToken := setupProfileHandlerTestRouter()

	t.Run("request change returns 200", func(t *testing.T) {
		body := `{"new_email":"new@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/email/request-change", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "new@example.com", data["new_email"])
	})

	t.Run("resend with active cooldown returns 429", func(t *testing.T) {
		body := `{"new_email":"new@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/email/resend-otp", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("verify change with correct OTP returns 200", func(t *testing.T) {
		// Set known OTP hash
		mac := hmac.New(sha256.New, []byte("01234567890123456789012345678901"))
		mac.Write([]byte("123456"))
		hash := hex.EncodeToString(mac.Sum(nil))

		_ = emailChangeRepo.Upsert(&domain.EmailChangeRequest{
			ID:           uuid.New(),
			UserID:       userID,
			NewEmail:     "verified@example.com",
			OTPHash:      hash,
			OTPExpiresAt: time.Now().Add(10 * time.Minute),
		})

		body := `{"new_email":"verified@example.com","otp":"123456"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/email/verify-change", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "verified@example.com", data["email"])
	})
}

func TestProfileHandler_AvatarEndpoints(t *testing.T) {
	router, _, _, _, _, validToken := setupProfileHandlerTestRouter()

	t.Run("upload multipart avatar returns 200", func(t *testing.T) {
		fileBytes, contentType := createTestPNGFile(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/avatar", bytes.NewReader(fileBytes))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(t, data["profile_image"])
	})

	t.Run("upload missing file returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/avatar", strings.NewReader(""))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=empty")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete avatar returns 200 with empty image", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/profile/avatar", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Empty(t, data["profile_image"])
	})
}

package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepoForAuthTest struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
	mu    sync.RWMutex
}

func (m *mockUserRepoForAuthTest) FindByID(id uuid.UUID) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func TestRequireActiveUser(t *testing.T) {
	middleware.ResetUserStatusCache()
	gin.SetMode(gin.TestMode)
	jwtService := token.NewJWTService("01234567890123456789012345678901")

	activeUserID := uuid.New()
	blockedUserID := uuid.New()

	activeToken, err := jwtService.GenerateAccessToken(activeUserID, "CUSTOMER")
	require.NoError(t, err)

	blockedToken, err := jwtService.GenerateAccessToken(blockedUserID, "CUSTOMER")
	require.NoError(t, err)

	repo := &mockUserRepoForAuthTest{
		users: map[uuid.UUID]*domain.User{
			activeUserID: {
				ID:     activeUserID,
				Role:   "CUSTOMER",
				Status: "ACTIVE",
			},
			blockedUserID: {
				ID:     blockedUserID,
				Role:   "CUSTOMER",
				Status: "BLOCKED",
			},
		},
	}

	router := gin.New()
	protected := router.Group("/api/protected")
	protected.Use(
		middleware.Auth(jwtService),
		middleware.RequireActiveUser(repo),
	)
	protected.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	t.Run("active user is allowed through", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/protected/ping", nil)
		req.Header.Set("Authorization", "Bearer "+activeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocked user is immediately rejected with 403 USER_BLOCKED", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/protected/ping", nil)
		req.Header.Set("Authorization", "Bearer "+blockedToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var body map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "USER_BLOCKED", body["code"])
	})

	t.Run("invalidating cache reflects status change immediately", func(t *testing.T) {
		// Change active user to blocked in repo, invalidate cache, verify rejection
		repo.mu.Lock()
		repo.users[activeUserID].Status = "BLOCKED"
		repo.mu.Unlock()

		middleware.InvalidateUserStatusCache(activeUserID)

		req, _ := http.NewRequest(http.MethodGet, "/api/protected/ping", nil)
		req.Header.Set("Authorization", "Bearer "+activeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		// Restore
		repo.mu.Lock()
		repo.users[activeUserID].Status = "ACTIVE"
		repo.mu.Unlock()
		middleware.InvalidateUserStatusCache(activeUserID)
	})
}

func TestUserStatusCache_CleanupExpired(t *testing.T) {
	middleware.ResetUserStatusCache()

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	now := time.Now()

	// user1 and user2 expired in the past
	middleware.SetUserStatusCacheForTest(user1, "ACTIVE", now.Add(-10*time.Second))
	middleware.SetUserStatusCacheForTest(user2, "BLOCKED", now.Add(-5*time.Second))
	// user3 is still active for 20 seconds
	middleware.SetUserStatusCacheForTest(user3, "ACTIVE", now.Add(20*time.Second))

	assert.Equal(t, 3, middleware.GetUserStatusCacheSize())

	// Run cleanup
	deleted := middleware.CleanupExpiredUserStatusCache(now)
	assert.Equal(t, 2, deleted)
	assert.Equal(t, 1, middleware.GetUserStatusCacheSize())

	// Advance time past user3's expiration
	deletedSecond := middleware.CleanupExpiredUserStatusCache(now.Add(30 * time.Second))
	assert.Equal(t, 1, deletedSecond)
	assert.Equal(t, 0, middleware.GetUserStatusCacheSize())
}

func TestUserStatusCache_ConcurrentAccess(t *testing.T) {
	middleware.ResetUserStatusCache()

	const numGoroutines = 40
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	testUserIDs := make([]uuid.UUID, 10)
	for i := range testUserIDs {
		testUserIDs[i] = uuid.New()
	}

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				u := testUserIDs[j%len(testUserIDs)]
				switch (routineID + j) % 4 {
				case 0:
					middleware.SetUserStatusCacheForTest(u, "ACTIVE", time.Now().Add(30*time.Second))
				case 1:
					middleware.InvalidateUserStatusCache(u)
				case 2:
					_ = middleware.CleanupExpiredUserStatusCache(time.Now())
				case 3:
					_ = middleware.GetUserStatusCacheSize()
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestStartUserStatusCacheCleaner(t *testing.T) {
	middleware.ResetUserStatusCache()

	user1 := uuid.New()
	middleware.SetUserStatusCacheForTest(user1, "ACTIVE", time.Now().Add(-1*time.Second))
	assert.Equal(t, 1, middleware.GetUserStatusCacheSize())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleaner with very short interval (20ms) for test
	middleware.StartUserStatusCacheCleaner(ctx, 20*time.Millisecond)

	// Wait up to 200ms for cleaner to run
	require.Eventually(t, func() bool {
		return middleware.GetUserStatusCacheSize() == 0
	}, 500*time.Millisecond, 20*time.Millisecond)
}

package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	userStatusCacheTTL        = 30 * time.Second
	userStatusCleanupInterval = 1 * time.Minute
	maxUserStatusCacheEntries = 10000
)

type userStatusCacheEntry struct {
	status    string
	expiresAt time.Time
}

var (
	userStatusCache       = make(map[uuid.UUID]userStatusCacheEntry)
	userStatusCacheMu     sync.RWMutex
	lastUserStatusCleanup time.Time
)

// InvalidateUserStatusCache clears any cached status for a user (called upon block/unblock).
func InvalidateUserStatusCache(userID uuid.UUID) {
	userStatusCacheMu.Lock()
	delete(userStatusCache, userID)
	userStatusCacheMu.Unlock()
}

// CleanupExpiredUserStatusCache removes expired entries from the cache to prevent memory growth.
// Returns the number of entries removed. Safe for concurrent use.
func CleanupExpiredUserStatusCache(now time.Time) int {
	userStatusCacheMu.Lock()
	defer userStatusCacheMu.Unlock()

	deletedCount := 0
	for id, entry := range userStatusCache {
		if now.After(entry.expiresAt) {
			delete(userStatusCache, id)
			deletedCount++
		}
	}
	lastUserStatusCleanup = now
	return deletedCount
}

// StartUserStatusCacheCleaner starts a background goroutine that periodically sweeps expired entries
// until the context is cancelled.
func StartUserStatusCacheCleaner(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = userStatusCleanupInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				CleanupExpiredUserStatusCache(now)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func Auth(jwtService *token.JWTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid authorization header",
			})
			return
		}

		claims, err := jwtService.ValidateAccessToken(parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid or expired token",
			})
			return
		}

		userIDString, ok := claims["userId"].(string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid token",
			})
			return
		}

		userID, err := uuid.Parse(userIDString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid token",
			})
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid token",
			})
			return
		}

		ctx.Set("user_id", userID)
		ctx.Set("user_role", role)

		ctx.Next()
	}
}

// RequireActiveUser enforces that the authenticated user is not BLOCKED (Bug #8).
// Uses a 30-second TTL cache with opportunistic and periodic expiration cleanup to prevent memory growth.
func RequireActiveUser(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		val, exists := ctx.Get("user_id")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "user id not found in context",
			})
			return
		}

		userID, ok := val.(uuid.UUID)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid user id in context",
			})
			return
		}

		now := time.Now()
		var status string
		var foundInCache bool

		userStatusCacheMu.RLock()
		if entry, ok := userStatusCache[userID]; ok && now.Before(entry.expiresAt) {
			status = entry.status
			foundInCache = true
		}
		userStatusCacheMu.RUnlock()

		if !foundInCache {
			user, err := userRepo.FindByID(userID)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    "UNAUTHORIZED",
					"message": "user not found",
				})
				return
			}
			status = user.Status

			userStatusCacheMu.Lock()
			// Opportunistic cleanup: prune expired entries if interval elapsed or cache exceeds threshold
			if now.Sub(lastUserStatusCleanup) >= userStatusCleanupInterval || len(userStatusCache) >= maxUserStatusCacheEntries {
				for id, entry := range userStatusCache {
					if now.After(entry.expiresAt) {
						delete(userStatusCache, id)
					}
				}
				lastUserStatusCleanup = now
			}

			userStatusCache[userID] = userStatusCacheEntry{
				status:    status,
				expiresAt: now.Add(userStatusCacheTTL),
			}
			userStatusCacheMu.Unlock()
		}

		if strings.ToUpper(status) == "BLOCKED" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "USER_BLOCKED",
				"message": "your account has been suspended by an administrator",
			})
			return
		}

		ctx.Next()
	}
}

package middleware

import (
	"time"

	"github.com/google/uuid"
)

// ResetUserStatusCache completely clears the cache (test-only export).
func ResetUserStatusCache() {
	userStatusCacheMu.Lock()
	userStatusCache = make(map[uuid.UUID]userStatusCacheEntry)
	lastUserStatusCleanup = time.Time{}
	userStatusCacheMu.Unlock()
}

// GetUserStatusCacheSize returns the current count of cached entries (test-only export).
func GetUserStatusCacheSize() int {
	userStatusCacheMu.RLock()
	defer userStatusCacheMu.RUnlock()
	return len(userStatusCache)
}

// SetUserStatusCacheForTest sets a cache entry with custom expiry for testing expiration behavior (test-only export).
func SetUserStatusCacheForTest(userID uuid.UUID, status string, expiresAt time.Time) {
	userStatusCacheMu.Lock()
	userStatusCache[userID] = userStatusCacheEntry{
		status:    status,
		expiresAt: expiresAt,
	}
	userStatusCacheMu.Unlock()
}

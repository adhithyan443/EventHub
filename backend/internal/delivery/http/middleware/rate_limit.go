package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientRateLimit struct {
	count       int
	windowStart time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientRateLimit
	maxRequests int
	window      time.Duration
	logger      *slog.Logger
}

func NewRateLimiter(
	maxRequests int,
	window time.Duration,
	logger *slog.Logger,
) *RateLimiter {
	return &RateLimiter{
		clients:     make(map[string]*clientRateLimit),
		maxRequests: maxRequests,
		window:      window,
		logger:      logger,
	}
}

func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := ctx.ClientIP()
		now := time.Now()

		r.mu.Lock()

		client, exists := r.clients[clientIP]

		if !exists || now.Sub(client.windowStart) >= r.window {
			r.clients[clientIP] = &clientRateLimit{
				count:       1,
				windowStart: now,
			}

			r.mu.Unlock()
			ctx.Next()
			return
		}

		if client.count >= r.maxRequests {
			r.mu.Unlock()

			requestID, _ := ctx.Get(RequestIDKey)

			r.logger.Warn(
				"auth_rate_limit_exceeded",
				"request_id", requestID,
				"method", ctx.Request.Method,
				"path", ctx.Request.URL.Path,
				"client_ip", clientIP,
			)

			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "too many requests, please try again later",
			})
			return
		}

		client.count++

		r.mu.Unlock()

		ctx.Next()
	}
}
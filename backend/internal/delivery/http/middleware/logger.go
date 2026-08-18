package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		start := time.Now()
		requestID, _ := ctx.Get(RequestIDKey)
		ctx.Next()

		logger.Info(
			"http_request",
			"request_id", requestID,
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"status", ctx.Writer.Status(),
			"latency", time.Since(start).String(),
		)
	}
}

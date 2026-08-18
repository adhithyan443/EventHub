package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error(
			"panic recovered",
			"error", recovered,
			"path", c.Request.URL.Path,
		)

		c.AbortWithStatusJSON(500, gin.H{
			"error": "Internal server error",
		})
	})
}

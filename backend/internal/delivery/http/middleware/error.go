package middleware

import (
	"errors"
	"log/slog"

	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/gin-gonic/gin"
)

func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		err := ctx.Errors.Last().Err

		var appErr *appErrors.AppError

		if errors.As(err, &appErr) {
			ctx.JSON(appErr.HTTPStatus, gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
			})
			return
		}

		requestID, _ := ctx.Get(RequestIDKey)
		logger.Error(
			"unhandled_server_error",
			"request_id", requestID,
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"error", err,
		)

		ctx.JSON(500, gin.H{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": "internal server error",
		})
	}
}

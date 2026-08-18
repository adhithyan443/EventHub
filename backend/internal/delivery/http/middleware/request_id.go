package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := uuid.NewString()

		ctx.Set(RequestIDKey, requestID)
		ctx.Header("X-Request-ID", requestID)

		ctx.Next()
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
				"message": "invalid authorized header",
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
				"message": "Invalid token",
			})
			return
		}

		ctx.Set("user_id", userID)
		ctx.Set("user_role", role)

		ctx.Next()
	}
}

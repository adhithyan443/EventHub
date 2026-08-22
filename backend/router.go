package main

import (
	"log/slog"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-gonic/gin"
)

func setupRouter(
	logger *slog.Logger,
	authHandler *handler.AuthHandler,
	jwtService *token.JWTService,
) *gin.Engine {

	router := gin.New()

	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.ErrorHandler(),
	)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	api := router.Group("/api/v1")

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	protected := api.Group("/protected")
	protected.Use(middleware.Auth(jwtService))

	protected.GET("/test", func(ctx *gin.Context) {
		userID, _ := ctx.Get("user_id")
		role, _ := ctx.Get("user_role")

		ctx.JSON(http.StatusOK, gin.H{
			"message": "you are authenticated",
			"user_id": userID,
			"role":    role,
		})
	})

	return router
}

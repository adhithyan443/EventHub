package main

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
)

func setupRouter(logger *slog.Logger, authHandler *handler.AuthHandler) *gin.Engine {
	router := gin.New()

	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.ErrorHandler(),
	)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	api := router.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)

	return router
}

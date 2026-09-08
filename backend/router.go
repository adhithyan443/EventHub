package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRouter(
	logger *slog.Logger,
	authHandler *handler.AuthHandler,
	organizerApplicationHandler *handler.OrganizerApplicationHandler,
	categoryHandler *handler.CategoryHandler,
	eventHandler *handler.EventHandler,
	jwtService *token.JWTService,
) *gin.Engine {
	router := gin.New()

	// Global middleware.
	router.Use(
		cors.New(cors.Config{
			AllowOrigins: []string{
				"http://localhost:5173",
				"http://127.0.0.1:5173",
			},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowHeaders: []string{
				"Origin",
				"Content-Type",
				"Accept",
				"Authorization",
			},
			AllowCredentials: true,
		}),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.ErrorHandler(logger),
	)

	registerHealthRoutes(router)
	registerAuthRoutes(router, authHandler, jwtService, logger)
	registerOrganizerRoutes(
		router,
		organizerApplicationHandler,
		eventHandler,
		jwtService,
	)
	registerAdminRoutes(router, organizerApplicationHandler, jwtService)
	registerCategoryRoutes(router, categoryHandler)

	return router
}

func registerHealthRoutes(router *gin.Engine) {
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
}

func registerAuthRoutes(
	router *gin.Engine,
	authHandler *handler.AuthHandler,
	jwtService *token.JWTService,
	logger *slog.Logger,
) {
	rateLimiter := middleware.NewRateLimiter(
		5,
		time.Minute,
		logger,
	)

	api := router.Group("/api/v1")
	auth := api.Group("/auth")

	// Public authentication endpoints.
	auth.POST(
		"/register",
		rateLimiter.Middleware(),
		authHandler.Register,
	)

	auth.POST(
		"/verify-otp",
		rateLimiter.Middleware(),
		authHandler.VerifyOTP,
	)

	auth.POST(
		"/resend-otp",
		rateLimiter.Middleware(),
		authHandler.ResendOTP,
	)

	auth.POST(
		"/login",
		rateLimiter.Middleware(),
		authHandler.Login,
	)

	auth.POST(
		"/forgot-password",
		rateLimiter.Middleware(),
		authHandler.ForgotPassword,
	)

	auth.POST(
		"/reset-password",
		authHandler.ResetPassword,
	)

	auth.GET(
		"/google",
		authHandler.GoogleLogin,
	)

	auth.GET(
		"/google/callback",
		authHandler.GoogleCallback,
	)

	// Token endpoints.
	auth.POST(
		"/refresh-token",
		authHandler.RefreshToken,
	)

	auth.POST(
		"/logout",
		middleware.Auth(jwtService),
		authHandler.Logout,
	)

	// Authenticated user endpoint.
	auth.GET(
		"/me",
		middleware.Auth(jwtService),
		authHandler.Me,
	)
}

func registerOrganizerRoutes(
	router *gin.Engine,
	organizerApplicationHandler *handler.OrganizerApplicationHandler,
	eventHandler *handler.EventHandler,
	jwtService *token.JWTService,
) {
	api := router.Group("/api/v1")

	organizer := api.Group("/organizers")
	organizer.Use(middleware.Auth(jwtService))

	// Customer organizer-application endpoints.
	organizer.POST(
		"/apply",
		organizerApplicationHandler.SubmitApplication,
	)

	organizer.GET(
		"/application",
		organizerApplicationHandler.GetMyApplication,
	)

	// Organizer event endpoints.
	organizer.POST(
		"/events",
		eventHandler.CreateEvent,
	)
}

func registerAdminRoutes(
	router *gin.Engine,
	organizerApplicationHandler *handler.OrganizerApplicationHandler,
	jwtService *token.JWTService,
) {
	api := router.Group("/api/v1")

	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(jwtService),
		middleware.RequireRole("ADMIN"),
	)

	admin.GET(
		"/organizer-applications",
		organizerApplicationHandler.ListApplications,
	)

	admin.GET(
		"/organizer-applications/:id",
		organizerApplicationHandler.GetApplication,
	)

	admin.PATCH(
		"/organizer-applications/:id/approve",
		organizerApplicationHandler.ApproveApplication,
	)

	admin.PATCH(
		"/organizer-applications/:id/reject",
		organizerApplicationHandler.RejectApplication,
	)
}

func registerCategoryRoutes(
	router *gin.Engine,
	categoryHandler *handler.CategoryHandler,
) {
	api := router.Group("/api/v1")

	api.GET(
		"/categories",
		categoryHandler.GetCategories,
	)
}

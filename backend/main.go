package main

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/config"
	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	appLogger "github.com/adhithyan443/EventHub/backend/internal/logger"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/service/email"
	"github.com/adhithyan443/EventHub/backend/internal/service/encryption"
	"github.com/adhithyan443/EventHub/backend/internal/service/google"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/joho/godotenv"
)

func main() {

	logger, closeLog, err := appLogger.New("app.log")
	if err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		return
	}
	defer closeLog()

	slog.SetDefault(logger)

	if err := godotenv.Load("../.env"); err != nil {
		logger.Warn(" .env file not found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		return
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		return
	}

	logger.Info("database connected successfully")

	if err := database.Migrate(db); err != nil {
		logger.Error("database migration failed", "error", err)
		return
	}
	logger.Info("database migration completed")

	if err := database.SeedCategories(db, logger); err != nil {
		logger.Error("category seeding failed", "error", err)
		return
	}

	logger.Info("category seeding completed")

	jwtService := token.NewJWTService(cfg.JWTSecret)

	userRepo := repository.NewUserRepository(db, logger)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	pendingRegistrationRepo := repository.NewPendingRegistrationRepository(db)
	txManager := repository.NewTransactionManager(db, logger)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)

	categoryRepository := repository.NewCategoryRepository(db, logger)
	// _ = repository.NewVenueRepository(db, logger)

	categoryUsecase := category.NewCategoryUsecase(categoryRepository, logger)

	categoryHandler := handler.NewCategoryHandler(categoryUsecase)

	encryptionService, err := encryption.NewService(cfg.EncryptionKey)
	if err != nil {
		logger.Error(
			"encryption_service_initialization_failed",
			"error", err,
		)
		return
	}

	googleOAuthService := google.NewGoogleOAuthService(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
	)

	emailService := email.NewSMTPEmailService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
		cfg.FrontendURL,
	)
	authUsecase := auth.NewAuthUsecase(
		userRepo,
		pendingRegistrationRepo,
		refreshTokenRepo,
		passwordResetTokenRepo,
		jwtService,
		txManager,
		emailService,
		googleOAuthService,
		logger,
	)

	authHandler := handler.NewAuthHandler(authUsecase, cfg.FrontendURL)

	// organizerApplicationRepo
	organizerApplicationRepo := repository.NewOrganizerApplicationRepository(
		db,
		logger,
	)

	organizerApplicationUsecase := organizer.NewApplicationUsecase(
		organizerApplicationRepo,
		encryptionService,
		logger,
	)

	adminApplicationUsecase := organizer.NewAdminApplicationUsecase(
		organizerApplicationRepo,
		txManager,
		logger,
	)

	organizerApplicationHandler := handler.NewOrganizerApplicationHandler(
		organizerApplicationUsecase,
		adminApplicationUsecase,
		logger,
	)

	router := setupRouter(logger, authHandler, organizerApplicationHandler, categoryHandler, jwtService)

	logger.Info(
		"EventHub backend started",
		"port", cfg.ServerPort,
		"environment", cfg.AppEnv,
	)

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		logger.Error("server failed to start", "error", err)
	}

	_ = slog.Default()
}

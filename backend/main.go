package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/adhithyan443/EventHub/backend/config"
	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	appLogger "github.com/adhithyan443/EventHub/backend/internal/logger"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/service/email"
	"github.com/adhithyan443/EventHub/backend/internal/service/encryption"
	"github.com/adhithyan443/EventHub/backend/internal/service/google"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	eventUC "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	logger, closeLog, err := appLogger.New("app.log")
	if err != nil {
		slog.Error("failed to initialize logger", "error", err)
		return
	}
	defer closeLog()

	slog.SetDefault(logger)

	if err := godotenv.Load("../.env"); err != nil {
		logger.Warn(".env file not found, using environment variables")
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

	s3Service, err := storage.NewS3Service(
		cfg,
		logger,
	)
	if err != nil {
		logger.Error(
			"s3_service_initialization_failed",
			"error", err,
		)
		return
	}

	jwtService := token.NewJWTService(cfg.JWTSecret)

	userRepo := repository.NewUserRepository(db, logger)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	pendingRegistrationRepo := repository.NewPendingRegistrationRepository(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)

	txManager := repository.NewTransactionManager(db, logger)

	reservationUsecase := reservation.NewReservationUsecase(
		txManager,
		logger,
	)

	reservationHandler := handler.NewReservationHandler(
		reservationUsecase,
		logger,
	)

	reservationExpiryWorker := worker.NewReservationExpiryWorker(
		reservationUsecase,
		logger,
		30*time.Second,
	)

	// Category dependencies.
	categoryRepository := repository.NewCategoryRepository(db, logger)

	categoryUsecase := category.NewCategoryUsecase(
		categoryRepository,
		logger,
	)

	categoryHandler := handler.NewCategoryHandler(
		categoryUsecase,
	)

	// Event dependencies.
	eventUsecase := eventUC.NewEventUsecase(
		txManager,
		s3Service,
		logger,
	)

	eventHandler := handler.NewEventHandler(
		eventUsecase,
		logger,
	)

	// Seat layout dependencies.
	seatLayoutUsecase := eventUC.NewSeatLayoutUsecase(
		txManager,
		logger,
	)

	seatLayoutHandler := handler.NewSeatLayoutHandler(
		seatLayoutUsecase,
		logger,
	)

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

	authHandler := handler.NewAuthHandler(
		authUsecase,
		cfg.FrontendURL,
	)

	organizerApplicationRepo := repository.NewOrganizerApplicationRepository(
		db,
		logger,
	)

	organizerRepo := repository.NewOrganizerRepository(
		db,
		logger,
	)

	organizerProfileRepo := repository.NewOrganizerProfileRepository(
		db,
		logger,
	)

	organizerAddressRepo := repository.NewOrganizerAddressRepository(
		db,
		logger,
	)

	organizerBankAccountRepo := repository.NewOrganizerBankAccountRepository(
		db,
		logger,
	)

	organizerProfileUsecase := organizer.NewProfileUsecase(
		organizerRepo,
		organizerProfileRepo,
		organizerAddressRepo,
		organizerBankAccountRepo,
		logger,
	)

	organizerProfileHandler := handler.NewOrganizerProfileHandler(
		organizerProfileUsecase,
		logger,
	)

	organizerApplicationUsecase := organizer.NewApplicationUsecase(
		organizerApplicationRepo,
		encryptionService,
		s3Service,
		logger,
	)

	adminApplicationUsecase := organizer.NewAdminApplicationUsecase(
		organizerApplicationRepo,
		txManager,
		s3Service,
		logger,
	)

	organizerApplicationHandler := handler.NewOrganizerApplicationHandler(
		organizerApplicationUsecase,
		adminApplicationUsecase,
		logger,
	)

	router := setupRouter(
		logger,
		authHandler,
		organizerApplicationHandler,
		organizerProfileHandler,
		categoryHandler,
		eventHandler,
		seatLayoutHandler,
		reservationHandler,
		jwtService,
	)

	logger.Info(
		"EventHub backend started",
		"port", cfg.ServerPort,
		"environment", cfg.AppEnv,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	var workerWG sync.WaitGroup

	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		reservationExpiryWorker.Run(ctx)
	}()

	go func() {
		logger.Info("HTTP server starting", "port", cfg.ServerPort)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful server shutdown failed", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("forced server close failed", "error", closeErr)
		}
	}

	stop()
	workerWG.Wait()
	logger.Info("EventHub backend stopped")
}

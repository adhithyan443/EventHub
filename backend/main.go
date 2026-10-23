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
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/checkout"
	eventUC "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/adhithyan443/EventHub/backend/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	// Application logging.
	logger, closeLog, err := appLogger.New("app.log")
	if err != nil {
		slog.Error("failed to initialize logger", "error", err)
		return
	}
	defer closeLog()

	slog.SetDefault(logger)

	// Configuration.
	if err := godotenv.Load("../.env"); err != nil {
		logger.Warn(".env file not found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		return
	}

	// Database setup.
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

	// Infrastructure services.
	s3Service, err := storage.NewS3Service(cfg, logger)
	if err != nil {
		logger.Error("s3_service_initialization_failed", "error", err)
		return
	}

	encryptionService, err := encryption.NewService(cfg.EncryptionKey)
	if err != nil {
		logger.Error("encryption_service_initialization_failed", "error", err)
		return
	}

	jwtService := token.NewJWTService(cfg.JWTSecret)

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

	// Shared repositories and transaction manager.
	userRepo := repository.NewUserRepository(db, logger)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	pendingRegistrationRepo := repository.NewPendingRegistrationRepository(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)

	txManager := repository.NewTransactionManager(db, logger)

	// Reservation dependencies.
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

	// Checkout dependencies.
	checkoutUsecase := checkout.NewCheckoutUsecase(
		txManager,
		logger,
	)

	checkoutHandler := handler.NewCheckoutHandler(
		checkoutUsecase,
		logger,
	)

	// Booking dependencies.
	bookingUsecase := booking.NewBookingUsecase(
		txManager,
		logger,
	)

	// Payment dependencies.
	razorpayProvider, err := payment.NewRazorpayProvider(
		cfg.RazorpayKeyID,
		cfg.RazorpayKeySecret,
		cfg.RazorpayWebhookSecret,
		logger,
	)
	if err != nil {
		logger.Error("razorpay provider initialization failed", "error", err)
		return
	}

	paymentUsecase := payment.NewPaymentUsecase(
		txManager,
		bookingUsecase,
		razorpayProvider,
		logger,
	)

	paymentHandler := handler.NewPaymentHandler(
		paymentUsecase,
		logger,
	)
	// paymentHandler := handler.NewPaymentHandler(paymentUsecase, logger)

	// Category dependencies.
	categoryRepository := repository.NewCategoryRepository(db, logger)

	categoryUsecase := category.NewCategoryUsecase(
		categoryRepository,
		logger,
	)

	categoryHandler := handler.NewCategoryHandler(categoryUsecase)

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

	// Authentication dependencies.
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

	// Organizer repositories.
	organizerApplicationRepo := repository.NewOrganizerApplicationRepository(
		db,
		logger,
	)

	organizerRepo := repository.NewOrganizerRepository(db, logger)
	organizerProfileRepo := repository.NewOrganizerProfileRepository(db, logger)
	organizerAddressRepo := repository.NewOrganizerAddressRepository(db, logger)
	organizerBankAccountRepo := repository.NewOrganizerBankAccountRepository(db, logger)

	// Organizer profile dependencies.
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

	// Organizer application dependencies.
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

	// Router setup.
	router := setupRouter(
		logger,
		authHandler,
		organizerApplicationHandler,
		organizerProfileHandler,
		categoryHandler,
		eventHandler,
		seatLayoutHandler,
		reservationHandler,
		checkoutHandler,
		paymentHandler,
		jwtService,
	)

	logger.Info(
		"EventHub backend started",
		"port", cfg.ServerPort,
		"environment", cfg.AppEnv,
	)

	// Graceful shutdown.
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

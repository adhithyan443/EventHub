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
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/middleware"
	appLogger "github.com/adhithyan443/EventHub/backend/internal/logger"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/service/email"
	"github.com/adhithyan443/EventHub/backend/internal/service/encryption"
	"github.com/adhithyan443/EventHub/backend/internal/service/google"
	"github.com/adhithyan443/EventHub/backend/internal/service/payout"
	"github.com/adhithyan443/EventHub/backend/internal/service/storage"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	admindashboard "github.com/adhithyan443/EventHub/backend/internal/usecase/admin_dashboard"
	adminledger "github.com/adhithyan443/EventHub/backend/internal/usecase/admin_ledger"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/booking"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/category"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/checkout"
	eventUC "github.com/adhithyan443/EventHub/backend/internal/usecase/events"
	financeusecase "github.com/adhithyan443/EventHub/backend/internal/usecase/finance"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/organizer"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/payment"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/pricingtax"
	profileUC "github.com/adhithyan443/EventHub/backend/internal/usecase/profile"
	refundUC "github.com/adhithyan443/EventHub/backend/internal/usecase/refund"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/reservation"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/settlement"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/ticket"
	userUC "github.com/adhithyan443/EventHub/backend/internal/usecase/user"
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

	if err := database.SeedFinanceAccounts(db, logger); err != nil {
		logger.Error("finance accounts seeding failed", "error", err)
		return
	}

	logger.Info("finance accounts seeding completed")

	qrService, err := ticket.NewQRService(cfg.QRSecret)
	if err != nil {
		logger.Error(
			"ticket QR service initialization failed",
			"error", err,
		)
		return
	}

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
	emailChangeRepo := repository.NewEmailChangeRequestRepository(db)

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

	// Payment provider.
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

	// Booking dependencies.
	bookingUsecase := booking.NewBookingUsecase(
		txManager,
		razorpayProvider,
		logger,
	)

	refundUsecase := refundUC.NewRefundUsecase(
		txManager,
		razorpayProvider,
		logger,
	)

	refundReconciliationWorker := worker.NewRefundReconciliationWorker(
		refundUsecase,
		logger,
		2*time.Minute,
		5*time.Minute,
	)

	bookingHandler := handler.NewBookingHandler(
		bookingUsecase,
		qrService,
		logger,
	)

	// Finance dependencies.
	financeUsecase, err := financeusecase.NewFinanceUsecase(
		txManager,
		logger,
	)
	if err != nil {
		logger.Error("finance usecase initialization failed", "error", err)
		return
	}
	refundUsecase.SetFinanceUsecase(financeUsecase)

	// Payment dependencies.
	paymentUsecase, err := payment.NewPaymentUsecase(
		txManager,
		bookingUsecase,
		financeUsecase,
		razorpayProvider,
		logger,
	)
	if err != nil {
		logger.Error("payment usecase initialization failed", "error", err)
		return
	}
	paymentUsecase.SetRefundProcessor(refundUsecase)

	paymentHandler := handler.NewPaymentHandler(
		paymentUsecase,
		cfg.RazorpayKeyID,
		logger,
	)
	// paymentHandler := handler.NewPaymentHandler(paymentUsecase, logger)

	// ticket
	ticketRepository := repository.NewTicketRepository(db, logger)
	eventRepository := repository.NewEventRepository(db, logger)
	eventScheduleRepository := repository.NewEventScheduleRepository(db, logger)
	venueRepository := repository.NewVenueRepository(db, logger)
	ticketTypeRepository := repository.NewTicketTypeRepository(db, logger)

	// Ticket dependencies.
	ticketUsecase := ticket.NewTicketUsecase(
		ticketRepository,
		eventRepository,
		eventScheduleRepository,
		userRepo,
		venueRepository,
		ticketTypeRepository,
		logger,
	)

	ticketHandler := handler.NewTicketHandler(ticketUsecase, qrService)
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
	eventUsecase.SetRefundProcessor(refundUsecase)

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
	authHandler.SetStorageService(s3Service)

	// Profile dependencies.
	profileUsecase := profileUC.NewProfileUsecase(
		userRepo,
		emailChangeRepo,
		pendingRegistrationRepo,
		txManager,
		emailService,
		s3Service,
		cfg.JWTSecret,
		logger,
	)
	profileHandler := handler.NewProfileHandler(
		profileUsecase,
		logger,
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
		s3Service,
		logger,
	)

	organizerProfileHandler := handler.NewOrganizerProfileHandler(
		organizerProfileUsecase,
		logger,
	)

	// Organizer application dependencies.
	organizerApplicationUsecase := organizer.NewApplicationUsecase(
		organizerApplicationRepo,
		organizerRepo,
		organizerProfileRepo,
		organizerAddressRepo,
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

	// Pricing and Tax configuration dependencies.
	pricingTaxUsecase := pricingtax.NewPricingTaxUsecase(
		txManager,
		logger,
	)

	adminPricingTaxHandler := handler.NewAdminPricingTaxHandler(
		pricingTaxUsecase,
		logger,
	)

	// Admin users and organizers dependencies.
	adminUserUsecase := userUC.NewAdminUserUsecase(userRepo, refreshTokenRepo, txManager, logger)
	adminUserHandler := handler.NewAdminUserHandler(adminUserUsecase, logger)

	adminOrganizerUsecase := organizer.NewAdminOrganizerUsecase(organizerRepo, s3Service, logger)
	adminOrganizerHandler := handler.NewAdminOrganizerHandler(adminOrganizerUsecase, logger)

	adminEventUsecase := eventUC.NewAdminEventUsecase(txManager, eventRepository, s3Service, logger)
	adminEventUsecase.SetRefundProcessor(refundUsecase)
	adminEventHandler := handler.NewAdminEventHandler(adminEventUsecase, logger)

	// Admin payments dependencies.
	paymentRepo := repository.NewPaymentRepository(db, logger)
	adminPaymentUsecase, err := payment.NewAdminPaymentUsecase(paymentRepo, logger)
	if err != nil {
		logger.Error("admin payment usecase initialization failed", "error", err)
		return
	}
	adminPaymentHandler := handler.NewAdminPaymentHandler(adminPaymentUsecase, logger)

	// Organizer payments dependencies.
	organizerPaymentUsecase, err := payment.NewOrganizerPaymentUsecase(
		paymentRepo,
		organizerRepo,
		eventRepository,
		logger,
	)
	if err != nil {
		logger.Error("organizer payment usecase initialization failed", "error", err)
		return
	}
	organizerPaymentHandler := handler.NewOrganizerPaymentHandler(organizerPaymentUsecase, logger)

	// Organizer settlement & balance projection dependencies.
	financeJournalRepo := repository.NewFinanceJournalRepository(db, logger)
	refundRepo := repository.NewRefundRepository(db, logger)
	settlementRepo := repository.NewSettlementRepository(db, logger)
	mockPayoutProvider := payout.NewMockPayoutProvider()

	organizerBalanceUsecase, err := settlement.NewOrganizerBalanceUsecase(
		financeJournalRepo,
		organizerRepo,
		eventRepository,
		organizerBankAccountRepo,
		refundRepo,
		logger,
	)
	if err != nil {
		logger.Error("organizer balance usecase initialization failed", "error", err)
		return
	}
	organizerBalanceUsecase.SetSettlementRepository(settlementRepo)

	settlementUsecase, err := settlement.NewSettlementUsecase(
		txManager,
		settlementRepo,
		organizerRepo,
		organizerBankAccountRepo,
		organizerBalanceUsecase,
		mockPayoutProvider,
		logger,
	)
	if err != nil {
		logger.Error("settlement usecase initialization failed", "error", err)
		return
	}

	organizerSettlementHandler := handler.NewOrganizerSettlementHandler(
		organizerBalanceUsecase,
		settlementUsecase,
		logger,
	)
	adminSettlementHandler := handler.NewAdminSettlementHandler(
		settlementUsecase,
		logger,
	)

	// Admin Dashboard dependencies.
	adminDashboardRepo := repository.NewAdminDashboardRepository(db, logger)
	adminDashboardUsecase := admindashboard.NewAdminDashboardUsecase(adminDashboardRepo, logger)
	adminDashboardHandler := handler.NewAdminDashboardHandler(adminDashboardUsecase, logger)

	// Admin Ledger dependencies.
	adminLedgerRepo := repository.NewAdminLedgerRepository(db, logger)
	adminLedgerUsecase := adminledger.NewAdminLedgerUsecase(adminLedgerRepo, logger)
	adminLedgerHandler := handler.NewAdminLedgerHandler(adminLedgerUsecase, logger)

	// Router setup.
	router := setupRouter(
		logger,
		authHandler,
		organizerApplicationHandler,
		organizerProfileHandler,
		organizerPaymentHandler,
		organizerSettlementHandler,
		categoryHandler,
		eventHandler,
		seatLayoutHandler,
		reservationHandler,
		checkoutHandler,
		bookingHandler,
		paymentHandler,
		ticketHandler,
		adminPricingTaxHandler,
		adminUserHandler,
		adminOrganizerHandler,
		adminEventHandler,
		adminPaymentHandler,
		adminSettlementHandler,
		adminDashboardHandler,
		adminLedgerHandler,
		profileHandler,
		jwtService,
		userRepo,
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

	dispatchTracker := worker.NewDispatchTracker()
	eventUsecase.SetDispatchTracker(dispatchTracker)
	adminEventUsecase.SetDispatchTracker(dispatchTracker)
	paymentUsecase.SetDispatchTracker(dispatchTracker)

	var workerWG sync.WaitGroup

	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		reservationExpiryWorker.Run(ctx)
	}()

	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		refundReconciliationWorker.Run(ctx)
	}()

	middleware.StartUserStatusCacheCleaner(ctx, 1*time.Minute)

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

	// Close tracker to atomically reject any new background dispatches from remaining handlers,
	// then wait for active background refund tasks within the remaining shutdown timeout.
	dispatchTracker.Close()
	refundDispatchDone := make(chan struct{})
	go func() {
		dispatchTracker.Wait()
		close(refundDispatchDone)
	}()

	select {
	case <-refundDispatchDone:
		logger.Info("all background refund tasks completed")
	case <-shutdownCtx.Done():
		logger.Warn("background refund tasks timed out during shutdown; remaining refunds will be processed by reconciliation worker")
	}

	stop()
	workerWG.Wait()

	logger.Info("EventHub backend stopped")
}

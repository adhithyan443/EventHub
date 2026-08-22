package main

import (
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/config"
	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	appLogger "github.com/adhithyan443/EventHub/backend/internal/logger"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/adhithyan443/EventHub/backend/internal/usecase/auth"
	"github.com/joho/godotenv"
)

func main() {

	logger := appLogger.New()

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

	jwtService := token.NewJWTService(cfg.JWTSecret)

	userRepo := repository.NewUserRepository(db)
	authUsecase := auth.NewAuthUsecase(userRepo, jwtService)
	authHandler := handler.NewAuthHandler(authUsecase)

	logger.Info("database migration completed")

	router := setupRouter(logger, authHandler, jwtService)

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

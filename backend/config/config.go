package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort  string
	AppEnv      string
	DatabaseURL string
	JWTSecret   string
	FrontendURL string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	EncryptionKey string
}

func Load() (Config, error) {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := requiredEnv("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	encryptionKey, err := requiredEnv("ENCRYPTION_KEY")
	if err != nil {
		return Config{}, err
	}

	smtpHost, err := requiredEnv("SMTP_HOST")
	if err != nil {
		return Config{}, err
	}

	smtpPort, err := requiredEnv("SMTP_PORT")
	if err != nil {
		return Config{}, err
	}

	smtpUsername, err := requiredEnv("SMTP_USERNAME")
	if err != nil {
		return Config{}, err
	}

	smtpPassword, err := requiredEnv("SMTP_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	smtpFrom, err := requiredEnv("SMTP_FROM")
	if err != nil {
		return Config{}, err
	}

	googleClientID, err := requiredEnv("GOOGLE_CLIENT_ID")
	if err != nil {
		return Config{}, err
	}

	googleClientSecret, err := requiredEnv("GOOGLE_CLIENT_SECRET")
	if err != nil {
		return Config{}, err
	}

	googleRedirectURL, err := requiredEnv("GOOGLE_REDIRECT_URL")
	if err != nil {
		return Config{}, err
	}

	return Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		AppEnv:      getEnv("APP_ENV", "development"),
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
		SMTPFrom:     smtpFrom,

		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		GoogleRedirectURL:  googleRedirectURL,

		EncryptionKey: encryptionKey,
	}, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is missing", key)
	}

	return value, nil
}

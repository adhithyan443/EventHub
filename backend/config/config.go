package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort  string
	AppEnv      string
	DatabaseURL string
}

func Load() (Config, error) {

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	return Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		AppEnv:      getEnv("APP_ENV", "development"),
		DatabaseURL: databaseURL,
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

package main

import (
	"fmt"

	"github.com/adhithyan443/EventHub/backend/config"
	"github.com/adhithyan443/EventHub/backend/internal/database"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load("../.env"); err != nil {
		fmt.Println("warning: .env file not found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("configuration error: %v\n", err)
		return
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("database connection failed: %v\n", err)
		return
	}
	_ = db

	fmt.Println("database connected successfully")

	router := setupRouter()

	fmt.Printf("EventHub backend started on port %s\n", cfg.ServerPort)
	fmt.Printf("Enviroment: %s\n", cfg.AppEnv)

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		fmt.Printf("server failed to start: %v \n", err)
	}
}

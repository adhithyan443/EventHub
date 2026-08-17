package main

import (
	"fmt"

	"github.com/adhithyan443/EventHub/backend/config"
)

func main() {

	cfg := config.Load()
	router := setupRouter()

	fmt.Printf("EventHub backend started on port %s\n", cfg.ServerPort)
	fmt.Printf("Enviroment: %s\n", cfg.AppEnv)

	if err := router.Run(":" + cfg.ServerPort); err != nil{
		fmt.Printf("server failed to start: %v \n",err)
	}
}

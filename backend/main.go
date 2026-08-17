package main

import (
	"fmt"

	"github.com/adhithyan443/EventHub/backend/config"
)

func main() {

	cfg := config.Load()

	fmt.Printf("EventHub backend started on port %s\n", cfg.ServerPort)
	fmt.Printf("Enviroment: %s\n", cfg.AppEnv)
}

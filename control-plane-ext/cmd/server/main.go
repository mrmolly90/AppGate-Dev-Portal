package main

import (
	"log"
	"os"

	"github.com/appgate/control-plane/internal/api"
	"github.com/appgate/control-plane/internal/config"
)

func main() {
	cfg := config.Load()

	server := api.NewServer(cfg)

	port := os.Getenv("CP_PORT")
	if port == "" {
		port = "8081"
	}

	if err := server.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

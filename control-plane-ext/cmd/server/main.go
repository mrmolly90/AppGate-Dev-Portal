package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/appgate/control-plane/internal/api"
	"github.com/appgate/control-plane/internal/config"
)

func main() {
	// Container healthcheck mode: exit 0 if /health responds OK
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		base := os.Getenv("CP_HEALTH_URL")
		if base == "" {
			base = "http://localhost:8081/health"
		}
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(base)
		if err != nil {
			log.Fatalf("control plane healthcheck failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("control plane healthcheck got HTTP %d", resp.StatusCode)
		}
		os.Exit(0)
	}

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

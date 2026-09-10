package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/appgate/dev-portal/apps/api/config"
	"github.com/appgate/dev-portal/apps/api/handlers"
	"github.com/appgate/dev-portal/apps/api/middleware"
	"github.com/appgate/dev-portal/apps/api/services"
	"github.com/appgate/dev-portal/pkg/controlplane"
	"github.com/rs/zerolog"
)

func main() {
	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize structured logger
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var logger zerolog.Logger
	if cfg.LogFormat == "json" {
		logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	} else {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Caller().Logger()
	}

	logger.Info().Str("bind", cfg.BindAddr).Str("cp_base_url", cfg.CPBaseURL).Msg("Dev-Portal BFF starting")

	// Initialize the control plane client
	cpOpts := []controlplane.ClientOption{
		controlplane.WithBaseURL(cfg.CPBaseURL),
		controlplane.WithTimeout(cfg.CPTimeout),
		controlplane.WithRetries(cfg.CPMaxRetries),
	}

	if cfg.CPAuthToken != "" {
		cpOpts = append(cpOpts, controlplane.WithBearerToken(cfg.CPAuthToken))
	} else if cfg.CPClientID != "" && cfg.CPClientSecret != "" {
		cpOpts = append(cpOpts, controlplane.WithAuthCredential(cfg.CPClientID, cfg.CPClientSecret))
	}

	if cfg.CPTLSInsecure {
		cpOpts = append(cpOpts, controlplane.WithInsecureSkipVerify(true))
	}

	cpClient, err := controlplane.New(cpOpts...)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create control plane client")
	}

	// Initialize services
	regSvc := services.NewRegistrationService(cpClient)
	statusSvc := services.NewStatusService(cpClient)

	// Initialize handlers
	registerHandler := handlers.NewRegisterHandler(regSvc, &logger)
	statusHandler := handlers.NewStatusHandler(statusSvc, &logger)
	healthHandler := handlers.NewHealthHandler(statusSvc, &logger)
	gatewaysHandler := handlers.NewGatewaysHandler(statusSvc, &logger)

	// Build middleware chain
	mux := http.NewServeMux()
	mux.Handle("/v1/tenant/register", registerHandler)
	mux.Handle("/v1/tenant/status/", statusHandler)
	mux.Handle("/v1/gateways", gatewaysHandler)
	mux.Handle("/health", healthHandler)

	// Apply middleware from outer to inner:
	// 1. Recovery (catches panics)
	// 2. Request ID injection
	// 3. Security headers
	// 4. CORS
	// 5. Logging
	// 6. Rate limiting
	// 7. Session auth

	rl := middleware.NewRateLimiter(
		cfg.APIRequestLimit,
		cfg.APIRequestBurst,
		cfg.APIRequestWindow,
		5*time.Minute,
		&logger,
	)

	var h http.Handler = mux
	h = middleware.Logging(&logger)(h)
	h = middleware.CORSMiddleware(cfg.AllowedOrigins, &logger)(h)
	h = middleware.SecurityHeaders(&logger)(h)
	h = middleware.RequestIDMiddleware(h)
	h = middleware.Recovery(&logger)(h)
	h = rl.RateLimit(h)

	// Optional session auth middleware (disabled by default for this stateless API)
	if cfg.SessionSecret != "" {
		sessionSecret := []byte(cfg.SessionSecret)
		h = middleware.EnforceSessionAuth(sessionSecret, &logger)(h)
	}

	// Configure TLS if enabled
	addr := cfg.BindAddr
	if addr == "" {
		addr = ":8080"
	}

	if cfg.TLSEnabled && cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
		logger.Info().Str("addr", addr).Msg("Starting HTTPS server")
		if err := http.ListenAndServeTLS(addr, cfg.TLSCertPath, cfg.TLSKeyPath, h); err != nil {
			logger.Fatal().Err(err).Msg("HTTPS server failed")
		}
	} else {
		logger.Info().Str("addr", addr).Msg("Starting HTTP server (use TLS in production)")
		if err := http.ListenAndServe(addr, h); err != nil {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}
}

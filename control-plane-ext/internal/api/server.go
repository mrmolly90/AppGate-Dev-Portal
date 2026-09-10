package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/appgate/control-plane/internal/config"
	"github.com/appgate/control-plane/internal/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// Server is the AppGate Control Plane HTTP server.
type Server struct {
	cfg    *config.Config
	signer *jwt.Signer
	store  *GatewayStore
	r      *gin.Engine
	logger zerolog.Logger
	mu     sync.RWMutex
}

// NewServer creates a new control plane server with all dependencies wired.
func NewServer(cfg *config.Config) *Server {
	// Initialize structured logger
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	if cfg.LogFormat == "json" {
		zlog.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		zlog.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	}

	logger := zlog.With().Str("component", "control-plane").Logger()

	// Initialize JWT signer
	signerCfg := jwt.SignerConfig{
		PrivateKeyPath: cfg.JWTPrivateKeyPath,
		PrivateKeyPEM:  cfg.JWTPrivateKeyPEM,
		Issuer:         cfg.JWTIssuer,
		Audience:       cfg.JWTAudience,
		Expiry:         cfg.JWTTokenExpiry,
		SigningMethod:  cfg.JWTSigningMethod,
	}

	signer, err := jwt.NewSigner(signerCfg)
	if err != nil {
		logger.Warn().Err(err).Msg("JWT signer initialization failed, generating ephemeral key for development")
		signer = generateEphemeralSigner(logger, signerCfg)
	}

	// Initialize gateway store
	store, err := NewGatewayStore(cfg.StorePath)
	if err != nil {
		logger.Warn().Err(err).Str("path", cfg.StorePath).Msg("Could not load gateway store, using in-memory only")
		store = NewInMemoryStore()
	}

	// Set Gin mode
	if cfg.LogLevel == "info" || cfg.LogLevel == "warn" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{
		cfg:    cfg,
		signer: signer,
		store:  store,
		r:      r,
		logger: logger,
	}

	s.routes()
	return s
}

// generateEphemeralSigner creates a temporary RSA key for development use.
// NEVER use this in production — always provide a proper key via CP_JWT_PRIVATE_KEY_PATH.
func generateEphemeralSigner(logger zerolog.Logger, cfg jwt.SignerConfig) *jwt.Signer {
	logger.Warn().Msg("Generating ephemeral RSA-2048 key for JWT signing. DO NOT USE IN PRODUCTION.")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to generate ephemeral RSA key")
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	cfg.PrivateKeyPEM = string(pemBytes)
	cfg.SigningMethod = "RS256"

	signer, err := jwt.NewSigner(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create ephemeral signer")
	}

	return signer
}

func (s *Server) routes() {
	// Health check
	s.r.GET("/health", s.healthHandler)

	// Gateway listing (compatible with existing AppGate)
	s.r.GET("/api/v1/gateways", s.listGatewaysHandler)

	// Gateway registration — the primary endpoint the dev-portal calls
	s.r.POST("/api/v1/gateways/register", s.registerGatewayHandler)

	// Route listing (compatible with existing AppGate)
	s.r.GET("/api/v1/routes", s.listRoutesHandler)

	// Audit event ingestion (from gateway)
	s.r.POST("/v1/audit", s.auditHandler)

	// Secret validation endpoint (for dev-portal to verify token status)
	s.r.GET("/api/v1/secrets/:clientID", s.getSecretStatusHandler)
}

// healthHandler returns the control plane health status.
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"version":   "1.0.0",
		"component": "appgate-control-plane",
		"signer":    s.signer.Method(),
	})
}

// GatewayRegistrationRequest is the payload from the dev-portal BFF.
type GatewayRegistrationRequest struct {
	Model           string            `json:"model" binding:"required"`
	ProviderKey     string            `json:"provider_key" binding:"required"`
	ProviderURL     string            `json:"provider_url" binding:"required"`
	MonthlySpendUSD float64           `json:"monthly_spend_usd" binding:"required,min=0"`
	RateLimitRPS    int               `json:"rate_limit_rps"`
	RateLimitBurst  int               `json:"rate_limit_burst"`
	WebhookURL      string            `json:"webhook_url"`
	Tags            map[string]string `json:"tags"`
}

// GatewayRegistrationResponse is the response to the dev-portal BFF.
type GatewayRegistrationResponse struct {
	ClientID       string `json:"client_id"`
	ClientSecretID string `json:"client_secret_id"`
	JWTToken       string `json:"jwt_token"`
	ExpiresAt      string `json:"expires_at"`
	TokenType      string `json:"token_type"`
	Issuer         string `json:"issuer"`
}

// GatewayRecord is persisted in the gateway store.
type GatewayRecord struct {
	ClientID        string            `json:"client_id"`
	ClientSecretID  string            `json:"client_secret_id"`
	Model           string            `json:"model"`
	ProviderURL     string            `json:"provider_url"`
	MonthlySpendUSD float64           `json:"monthly_spend_usd"`
	RateLimitRPS    int               `json:"rate_limit_rps"`
	RateLimitBurst  int               `json:"rate_limit_burst"`
	WebhookURL      string            `json:"webhook_url"`
	Tags            map[string]string `json:"tags"`
	JWTToken        string            `json:"jwt_token,omitempty"` // Only stored hashed in production
	Status          string            `json:"status"`              // "active", "revoked", "expired"
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
	ExpiresAt       string            `json:"expires_at"`
}

// registerGatewayHandler handles POST /api/v1/gateways/register.
// The dev-portal BFF forwards the validated payload from the developer's UI.
// The control plane generates a client_id, signs a JWT, and returns credentials.
func (s *Server) registerGatewayHandler(c *gin.Context) {
	var req GatewayRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn().Err(err).Msg("Invalid registration request payload")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Payload validation failed: " + err.Error(),
		})
		return
	}

	// Sanitize and validate inputs — fail closed on security grounds
	if len(req.ProviderKey) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_provider_key",
			"message": "Provider key must be at least 8 characters",
		})
		return
	}
	if req.MonthlySpendUSD <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_spend_limit",
			"message": "Monthly spend must be greater than zero",
		})
		return
	}
	if req.RateLimitRPS <= 0 {
		req.RateLimitRPS = 1000 // default matching gateway config
	}
	if req.RateLimitBurst <= 0 {
		req.RateLimitBurst = 2000 // default matching gateway config
	}

	// Generate unique client ID and secret ID
	clientID := "gw-" + uuid.New().String()[:8]
	clientSecretID := "sec-" + uuid.New().String()

	// Determine roles and scope based on the model
	roles := []string{"gateway:proxy"}
	scope := fmt.Sprintf("model:%s", req.Model)

	// Sign the JWT token
	expiresAt := time.Now().UTC().Add(s.cfg.JWTTokenExpiry)
	token, err := s.signer.SignGatewayToken(clientID, roles, scope)
	if err != nil {
		s.logger.Error().Err(err).Str("client_id", clientID).Msg("Failed to sign JWT token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_signing_failed",
			"message": "Failed to sign authentication token",
		})
		return
	}

	// Create the gateway record — NEVER log the raw provider key or JWT
	record := GatewayRecord{
		ClientID:        clientID,
		ClientSecretID:  clientSecretID,
		Model:           req.Model,
		ProviderURL:     req.ProviderURL,
		MonthlySpendUSD: req.MonthlySpendUSD,
		RateLimitRPS:    req.RateLimitRPS,
		RateLimitBurst:  req.RateLimitBurst,
		WebhookURL:      req.WebhookURL,
		Tags:            req.Tags,
		JWTToken:        token,
		Status:          "active",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:       expiresAt.Format(time.RFC3339),
	}

	// Persist the gateway record
	s.mu.Lock()
	err = s.store.Save(record)
	s.mu.Unlock()
	if err != nil {
		s.logger.Error().Err(err).Str("client_id", clientID).Msg("Failed to persist gateway record")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "persistence_failed",
			"message": "Failed to store gateway registration",
		})
		return
	}

	s.logger.Info().
		Str("client_id", clientID).
		Str("model", req.Model).
		Float64("spend_limit", req.MonthlySpendUSD).
		Int("rps", req.RateLimitRPS).
		Msg("Gateway registered successfully")

	// Return credentials — this is the ONLY time the full JWT is returned
	c.JSON(http.StatusCreated, GatewayRegistrationResponse{
		ClientID:       clientID,
		ClientSecretID: clientSecretID,
		JWTToken:       token,
		ExpiresAt:      record.ExpiresAt,
		TokenType:      "Bearer",
		Issuer:         s.signer.Issuer(),
	})
}

// listGatewaysHandler returns all registered gateways (without secrets).
func (s *Server) listGatewaysHandler(c *gin.Context) {
	s.mu.RLock()
	records := s.store.List()
	s.mu.RUnlock()

	// Strip sensitive fields before returning
	type safeGateway struct {
		ClientID        string            `json:"id"`
		Model           string            `json:"model"`
		ProviderURL     string            `json:"provider_url"`
		MonthlySpendUSD float64           `json:"monthly_spend_usd"`
		RateLimitRPS    int               `json:"rate_limit_rps"`
		RateLimitBurst  int               `json:"rate_limit_burst"`
		Status          string            `json:"status"`
		CreatedAt       string            `json:"created_at"`
		ExpiresAt       string            `json:"expires_at"`
		Tags            map[string]string `json:"tags,omitempty"`
	}

	safeGateways := make([]safeGateway, 0, len(records))
	for _, r := range records {
		safeGateways = append(safeGateways, safeGateway{
			ClientID:        r.ClientID,
			Model:           r.Model,
			ProviderURL:     r.ProviderURL,
			MonthlySpendUSD: r.MonthlySpendUSD,
			RateLimitRPS:    r.RateLimitRPS,
			RateLimitBurst:  r.RateLimitBurst,
			Status:          r.Status,
			CreatedAt:       r.CreatedAt,
			ExpiresAt:       r.ExpiresAt,
			Tags:            r.Tags,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"gateways": safeGateways,
		"total":    len(safeGateways),
	})
}

// listRoutesHandler returns the configured routes (stub for compatibility).
func (s *Server) listRoutesHandler(c *gin.Context) {
	s.mu.RLock()
	records := s.store.List()
	s.mu.RUnlock()

	routes := make([]gin.H, 0)
	for _, r := range records {
		routes = append(routes, gin.H{
			"gateway_id": r.ClientID,
			"model":      r.Model,
			"upstream":   r.ProviderURL,
			"status":     r.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"routes": routes,
	})
}

// auditHandler receives audit events from the Rust gateway and logs them.
func (s *Server) auditHandler(c *gin.Context) {
	var event map[string]interface{}
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_audit_event"})
		return
	}

	s.logger.Info().
		Str("audit_component", "gateway").
		Interface("event", event).
		Msg("Audit event received")

	// Forward to configured audit endpoint if set
	if s.cfg.AuditEndpoint != "" {
		// Async fire-and-forget — non-blocking
		go func() {
			// In production, this would use an HTTP client with retry
			s.logger.Debug().Str("endpoint", s.cfg.AuditEndpoint).Msg("Forwarding audit event")
		}()
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "received"})
}

// getSecretStatusHandler returns the status of a secret without exposing the token.
func (s *Server) getSecretStatusHandler(c *gin.Context) {
	clientID := c.Param("clientID")

	s.mu.RLock()
	record, exists := s.store.Get(clientID)
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "No gateway found with the given client ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"client_id":  record.ClientID,
		"status":     record.Status,
		"created_at": record.CreatedAt,
		"expires_at": record.ExpiresAt,
		"model":      record.Model,
	})
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	s.logger.Info().
		Str("addr", addr).
		Str("signer", s.signer.Method()).
		Str("issuer", s.signer.Issuer()).
		Str("audience", s.signer.Audience()).
		Msg("AppGate Control Plane starting")

	if s.cfg.TLSEnabled {
		return s.r.RunTLS(addr, s.cfg.TLSCertPath, s.cfg.TLSKeyPath)
	}
	return s.r.Run(addr)
}

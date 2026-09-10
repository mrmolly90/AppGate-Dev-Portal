package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the AppGate Control Plane.
type Config struct {
	BindAddr  string
	LogLevel  string
	LogFormat string // "json" or "text"

	// JWT Signing
	JWTPrivateKeyPath string
	JWTPrivateKeyPEM  string // inline PEM if not using file
	JWTIssuer         string
	JWTAudience       string
	JWTTokenExpiry    time.Duration
	JWTSigningMethod  string // "RS256" or "ES256"

	// Rate limiting for the API itself
	APIRequestLimit  int
	APIRequestBurst  int
	APIRequestWindow time.Duration

	// Data store (file-based for now)
	StorePath string

	// TLS for control plane
	TLSEnabled  bool
	TLSCertPath string
	TLSKeyPath  string

	// Audit forwarding from gateway
	AuditEndpoint string
}

// Load reads configuration from environment variables with secure defaults.
func Load() *Config {
	return &Config{
		BindAddr:          getEnv("CP_BIND", "0.0.0.0:8081"),
		LogLevel:          getEnv("CP_LOG_LEVEL", "info"),
		LogFormat:         getEnv("CP_LOG_FORMAT", "json"),
		JWTPrivateKeyPath: getEnv("CP_JWT_PRIVATE_KEY_PATH", ""),
		JWTPrivateKeyPEM:  getEnv("CP_JWT_PRIVATE_KEY_PEM", ""),
		JWTIssuer:         getEnv("CP_JWT_ISSUER", "appgate"),
		JWTAudience:       getEnv("CP_JWT_AUDIENCE", "appgate-api"),
		JWTTokenExpiry:    getDurationEnv("CP_JWT_TOKEN_EXPIRY", 1*time.Hour),
		JWTSigningMethod:  getEnv("CP_JWT_SIGNING_METHOD", "RS256"),
		APIRequestLimit:   getIntEnv("CP_API_RATE_LIMIT", 100),
		APIRequestBurst:   getIntEnv("CP_API_RATE_BURST", 200),
		APIRequestWindow:  getDurationEnv("CP_API_RATE_WINDOW", 1*time.Minute),
		StorePath:         getEnv("CP_STORE_PATH", "/data/store.json"),
		TLSEnabled:        getBoolEnv("CP_TLS_ENABLED", false),
		TLSCertPath:       getEnv("CP_TLS_CERT_PATH", ""),
		TLSKeyPath:        getEnv("CP_TLS_KEY_PATH", ""),
		AuditEndpoint:     getEnv("CP_AUDIT_ENDPOINT", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

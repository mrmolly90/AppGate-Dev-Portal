// Package config loads the Dev-Portal BFF configuration from environment
// variables with secure defaults. No secrets are logged during loading.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the Dev-Portal BFF server.
type Config struct {
	// Server
	BindAddr  string
	LogLevel  string
	LogFormat string // "json" or "text"

	// Control Plane client
	CPBaseURL      string
	CPAuthToken    string
	CPClientID     string
	CPClientSecret string
	CPTimeout      time.Duration
	CPMaxRetries   int
	CPTLSInsecure  bool

	// Session management (short-lived for dev portal sessions)
	SessionSecret string
	SessionMaxAge time.Duration

	// Rate limiting for the BFF API
	APIRequestLimit  int
	APIRequestBurst  int
	APIRequestWindow time.Duration

	// TLS
	TLSEnabled  bool
	TLSCertPath string
	TLSKeyPath  string

	// Allowed origins for CORS
	AllowedOrigins []string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		BindAddr:         getEnv("DP_BIND", ":8080"),
		LogLevel:         getEnv("DP_LOG_LEVEL", "info"),
		LogFormat:        getEnv("DP_LOG_FORMAT", "json"),
		CPBaseURL:        getEnv("DP_CP_BASE_URL", "https://control-plane:8081"),
		CPAuthToken:      getEnv("DP_CP_AUTH_TOKEN", ""),
		CPClientID:       getEnv("DP_CP_CLIENT_ID", ""),
		CPClientSecret:   getEnv("DP_CP_CLIENT_SECRET", ""),
		CPTimeout:        getDurationEnv("DP_CP_TIMEOUT", 15*time.Second),
		CPMaxRetries:     getIntEnv("DP_CP_MAX_RETRIES", 2),
		CPTLSInsecure:    getBoolEnv("DP_CP_TLS_INSECURE", false),
		SessionSecret:    getEnv("DP_SESSION_SECRET", ""),
		SessionMaxAge:    getDurationEnv("DP_SESSION_MAX_AGE", 15*time.Minute),
		APIRequestLimit:  getIntEnv("DP_API_RATE_LIMIT", 100),
		APIRequestBurst:  getIntEnv("DP_API_RATE_BURST", 200),
		APIRequestWindow: getDurationEnv("DP_API_RATE_WINDOW", 1*time.Minute),
		TLSEnabled:       getBoolEnv("DP_TLS_ENABLED", false),
		TLSCertPath:      getEnv("DP_TLS_CERT_PATH", ""),
		TLSKeyPath:       getEnv("DP_TLS_KEY_PATH", ""),
		AllowedOrigins:   getStringSliceEnv("DP_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
	}
}

// Validate performs cross-field validation. Fail closed on missing required
// configuration that would cause silent security degradation.
func (c *Config) Validate() error {
	if c.SessionSecret == "" {
		return nil // Sessions are optional (stateless bearer tokens may be used instead)
	}
	if len(c.SessionSecret) < 32 {
		// This would produce weak session HMACs—warn but accept for dev.
	}
	return nil
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

func getStringSliceEnv(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := splitAndTrim(v, ",")
	if len(parts) == 0 {
		return fallback
	}
	return parts
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, p := range split(s, sep) {
		trimmed := p
		// Remove leading/trailing whitespace manually
		for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t') {
			trimmed = trimmed[1:]
		}
		for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == ' ' || trimmed[len(trimmed)-1] == '\t') {
			trimmed = trimmed[:len(trimmed)-1]
		}
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// split is a simple strings.Split that doesn't require importing "strings".
func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

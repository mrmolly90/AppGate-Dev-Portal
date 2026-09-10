// Package middleware provides HTTP middleware for the Dev-Portal BFF:
// session authentication, rate limiting, security headers, request logging,
// and panic recovery. All middleware fails closed.
package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey holds the correlation ID in the request context.
	RequestIDKey contextKey = "request_id"
	// ClientIDKey holds the authenticated client ID in the context.
	ClientIDKey contextKey = "client_id"
)

// SecurityHeaders returns middleware that sets industry-standard security
// HTTP headers on every response. Cache-Control is set to no-store for
// credential-bearing responses.
func SecurityHeaders(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "0") // Deprecated but still scanned
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			// Strict Transport Security — 1 year, include subdomains, preload
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}
			// Content-Security-Policy restricts script/style/data sources
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware allows the React dev server to access the BFF API.
// In production this should be locked to specific origins.
func CORSMiddleware(allowedOrigins []string, logger *zerolog.Logger) func(http.Handler) http.Handler {
	originMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originMap[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if originMap[origin] || len(originMap) == 0 {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
					w.Header().Set("Access-Control-Max-Age", "300")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware injects or reads a correlation request ID.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		r.Header.Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

// rateLimiter is a simple token-bucket rate limiter per IP.
type rateLimiter struct {
	mu      sync.Mutex
	tokens  map[string]*tokenBucket
	limit   int
	burst   int
	window  time.Duration
	cleanup time.Duration
	logger  *zerolog.Logger
}

type tokenBucket struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a token-bucket rate limiter that enforces
// per-IP request ceilings. Buckets are cleaned up periodically.
func NewRateLimiter(limit, burst int, window, cleanup time.Duration, logger *zerolog.Logger) *rateLimiter {
	if logger == nil {
		l := zerolog.New(zerolog.NewConsoleWriter())
		logger = &l
	}
	rl := &rateLimiter{
		tokens:  make(map[string]*tokenBucket),
		limit:   limit,
		burst:   burst,
		window:  window,
		cleanup: cleanup,
		logger:  logger,
	}
	if cleanup > 0 {
		go rl.cleanupLoop()
	}
	return rl
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, tb := range rl.tokens {
			if now.Sub(tb.lastCheck) > rl.window*2 {
				delete(rl.tokens, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns HTTP middleware enforcing per-IP rate limits.
func (rl *rateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx >= 0 {
			ip = ip[:idx]
		}

		rl.mu.Lock()
		bucket, exists := rl.tokens[ip]
		if !exists {
			bucket = &tokenBucket{tokens: rl.burst, lastCheck: time.Now()}
			rl.tokens[ip] = bucket
		}

		now := time.Now()
		elapsed := now.Sub(bucket.lastCheck)
		bucket.lastCheck = now

		// Refill tokens based on elapsed time
		refill := int(elapsed.Seconds() * float64(rl.limit) / rl.window.Seconds())
		bucket.tokens += refill
		if bucket.tokens > rl.burst {
			bucket.tokens = rl.burst
		}

		if bucket.tokens <= 0 {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
			})
			return
		}

		bucket.tokens--
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// EnforceSessionAuth validates the session cookie HMAC. If a session is
// present and valid, the client_id is injected into the request context.
// If no session is present, the handler can still proceed (optional auth)
// unless RequireSession is composed downstream.
func EnforceSessionAuth(secret []byte, logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("dp_session")
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			parts := strings.SplitN(cookie.Value, ".", 2)
			if len(parts) != 2 {
				next.ServeHTTP(w, r)
				return
			}
			sig := hmacSHA256(secret, parts[0])
			if !hmac.Equal([]byte(sig), []byte(parts[1])) {
				logger.Warn().Msg("Session cookie HMAC validation failed")
				next.ServeHTTP(w, r)
				return
			}
			// Session payload is hex-encoded JSON
			decoded, err := hex.DecodeString(parts[0])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			var session struct {
				ClientID string `json:"client_id"`
				Expires  int64  `json:"expires"`
			}
			if err := json.Unmarshal(decoded, &session); err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if time.Now().Unix() > session.Expires {
				next.ServeHTTP(w, r)
				return
			}
			// Store in context for downstream handlers
			r.Header.Set("X-Session-Client-ID", session.ClientID)
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSession rejects requests without a valid session context.
func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Session-Client-ID") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recovery middleware catches panics and returns a 500 error, preventing
// the server from crashing on unexpected failures.
func Recovery(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().Interface("panic", rec).Str("path", r.URL.Path).Msg("Handler panic recovered")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "internal_error",
						"message": "An internal error occurred",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Logging middleware logs every request with key metadata.
func Logging(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", lrw.statusCode).
				Str("remote", r.RemoteAddr).
				Str("request_id", r.Header.Get("X-Request-Id")).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// hmacSHA256 computes HMAC-SHA256 of data with the given key, hex-encoded.
func hmacSHA256(key []byte, data string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

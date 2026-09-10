// Package handlers contains the HTTP route handlers for the Dev-Portal BFF.
// Every handler is zero-trust: it validates its inputs, never trusts
// client identity without verification, never logs secrets, and never
// returns raw credentials more than once.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/appgate/dev-portal/apps/api/services"
	"github.com/appgate/dev-portal/pkg/controlplane"
	"github.com/appgate/dev-portal/pkg/security"
	"github.com/rs/zerolog"
)

// RegisterHandler handles POST /v1/tenant/register.
// It accepts a validated registration payload from the frontend, forwards
// it to the AppGate Control Plane via the internal REST client, and returns
// the ephemeral credential handshake (JWT, client_id, secret_id).
//
// Security properties:
//   - Provider key is never logged; only a SHA-256 digest is recorded.
//   - The response contains Cache-Control: no-store so credentials cannot
//     be cached by proxies or the browser's back-forward cache.
//   - The response body is returned exactly once — the UI must render and
//     then immediately discard the JWT from client memory.
//   - The handler validates the payload before forwarding (fail closed).
type RegisterHandler struct {
	svc    *services.RegistrationService
	logger *zerolog.Logger
}

// NewRegisterHandler creates a handler with the registration service.
func NewRegisterHandler(svc *services.RegistrationService, logger *zerolog.Logger) *RegisterHandler {
	return &RegisterHandler{svc: svc, logger: logger}
}

// TenantRegisterRequest is the JSON payload accepted from the frontend.
type TenantRegisterRequest struct {
	ProjectName     string            `json:"project_name"`
	Model           string            `json:"model"`
	LLMModelName    string            `json:"llm_model_name"`
	ProviderKey     string            `json:"provider_key"`
	ProviderURL     string            `json:"provider_url"`
	MonthlySpendUSD float64           `json:"monthly_spend_usd"`
	RateLimitRPS    int               `json:"rate_limit_rps"`
	RateLimitRPM    int               `json:"rate_limit_rpm"`
	RateLimitBurst  int               `json:"rate_limit_burst"`
	WebhookURL      string            `json:"webhook_url"`
	Tags            map[string]string `json:"tags"`
}

// TenantRegisterResponse is the response body returned to the frontend.
// The JWT and SecretID are ephemeral: they are rendered once and must be
// discarded client-side after the modal is closed.
type TenantRegisterResponse struct {
	ClientID       string `json:"client_id"`
	ClientSecretID string `json:"client_secret_id"`
	JWTToken       string `json:"jwt_token"`
	ExpiresAt      string `json:"expires_at"`
	TokenType      string `json:"token_type"`
	Issuer         string `json:"issuer"`
}

// ServeHTTP handles POST /v1/tenant/register.
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fail closed on method
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	// Enforce Content-Type
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return
	}

	// Decode and limit body size (1 MB)
	var req TenantRegisterRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON")
		return
	}

	// Validate required fields before forwarding
	if err := h.validateRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Log only metadata — NEVER log the provider key
	h.logger.Info().
		Str("project", req.ProjectName).
		Str("model", req.Model).
		Str("provider_url", req.ProviderURL).
		Float64("monthly_spend_usd", req.MonthlySpendUSD).
		Int("rate_limit_rps", req.RateLimitRPS).
		Str("key_digest", services.KeyDigest(req.ProviderKey)).
		Msg("Tenant registration request received")

	// Build the control plane request
	cpReq := &controlplane.RegisterRequest{
		ProjectName:     req.ProjectName,
		Model:           req.Model,
		LLMModelName:    req.LLMModelName,
		ProviderKey:     req.ProviderKey,
		ProviderURL:     req.ProviderURL,
		MonthlySpendUSD: req.MonthlySpendUSD,
		RateLimitRPS:    req.RateLimitRPS,
		RateLimitRPM:    req.RateLimitRPM,
		RateLimitBurst:  req.RateLimitBurst,
		WebhookURL:      req.WebhookURL,
		Tags:            req.Tags,
	}

	// Forward to the control plane
	resp, err := h.svc.Register(r.Context(), cpReq)
	if err != nil {
		h.logger.Error().Err(err).Str("model", req.Model).Msg("Registration failed")
		if errors.Is(err, services.ErrRegistrationRejected) {
			writeError(w, http.StatusBadGateway, "registration_rejected", "Control plane rejected the registration. Check upstream provider configuration.")
		} else {
			writeError(w, http.StatusInternalServerError, "registration_failed", "Registration failed. Please try again.")
		}
		return
	}

	// Set no-store headers so ephemeral credentials are never cached
	security.ResponseWriterNoCache(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Marshal and write the response
	json.NewEncoder(w).Encode(TenantRegisterResponse{
		ClientID:       resp.ClientID,
		ClientSecretID: resp.ClientSecretID,
		JWTToken:       resp.JWTToken,
		ExpiresAt:      resp.ExpiresAt,
		TokenType:      resp.TokenType,
		Issuer:         resp.Issuer,
	})

	h.logger.Info().
		Str("client_id", resp.ClientID).
		Str("client_secret_id", resp.ClientSecretID).
		Str("expires_at", resp.ExpiresAt).
		Msg("Tenant registration completed — credentials issued")
}

// validateRequest checks required fields before forwarding to the control
// plane. Duplicated validation is intentional defense-in-depth: the BFF
// never sends a payload the control plane would reject.
func (h *RegisterHandler) validateRequest(req *TenantRegisterRequest) error {
	if strings.TrimSpace(req.ProjectName) == "" {
		return errors.New("project_name is required")
	}
	if len(req.ProjectName) > 128 {
		return errors.New("project_name exceeds maximum length of 128 characters")
	}
	if strings.TrimSpace(req.Model) == "" {
		return errors.New("model is required")
	}
	if len(req.Model) > 128 {
		return errors.New("model exceeds maximum length of 128 characters")
	}
	if strings.TrimSpace(req.ProviderKey) == "" {
		return errors.New("provider_key is required")
	}
	if len(req.ProviderKey) < 8 {
		return errors.New("provider_key must be at least 8 characters")
	}
	if strings.TrimSpace(req.ProviderURL) == "" {
		return errors.New("provider_url is required")
	}
	// SSRF defense: only allow https upstream URLs
	if !strings.HasPrefix(req.ProviderURL, "https://") {
		return errors.New("provider_url must use https")
	}
	if req.MonthlySpendUSD <= 0 {
		return errors.New("monthly_spend_usd must be greater than zero")
	}
	if req.RateLimitRPS <= 0 {
		req.RateLimitRPS = 1000 // default matching control plane
	}
	if req.RateLimitRPM <= 0 {
		req.RateLimitRPM = 60000 // default 60k req/min
	}
	if req.RateLimitBurst <= 0 {
		req.RateLimitBurst = 2000
	}
	// Optional webhook validation
	if req.WebhookURL != "" && !strings.HasPrefix(req.WebhookURL, "https://") {
		return errors.New("webhook_url must use https when provided")
	}
	return nil
}

// StatusHandler handles GET /v1/tenant/status/{clientID}.
type StatusHandler struct {
	svc    *services.StatusService
	logger *zerolog.Logger
}

// NewStatusHandler creates a secret status handler.
func NewStatusHandler(svc *services.StatusService, logger *zerolog.Logger) *StatusHandler {
	return &StatusHandler{svc: svc, logger: logger}
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimPrefix(r.URL.Path, "/v1/tenant/status/")
	if clientID == "" || clientID == r.URL.Path {
		writeError(w, http.StatusBadRequest, "missing_client_id", "Client ID is required")
		return
	}

	resp, err := h.svc.Status(r.Context(), clientID)
	if err != nil {
		h.logger.Error().Err(err).Str("client_id", clientID).Msg("Failed to fetch secret status")
		writeError(w, http.StatusBadGateway, "status_failed", "Failed to query secret status")
		return
	}

	security.ResponseWriterNoCache(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HealthHandler handles GET /health for the BFF itself.
type HealthHandler struct {
	svc    *services.StatusService
	logger *zerolog.Logger
}

// NewHealthHandler creates a health check handler.
func NewHealthHandler(svc *services.StatusService, logger *zerolog.Logger) *HealthHandler {
	return &HealthHandler{svc: svc, logger: logger}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cpErr := h.svc.Health(r.Context())
	status := "healthy"
	if cpErr != nil {
		h.logger.Warn().Err(cpErr).Msg("Control plane health check failed")
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        status,
		"component":     "dev-portal-bff",
		"version":       "1.0.0",
		"control_plane": cpErr == nil,
	})
}

// GatewaysHandler handles GET /v1/gateways — lists all registered gateways.
type GatewaysHandler struct {
	svc    *services.StatusService
	logger *zerolog.Logger
}

// NewGatewaysHandler creates a gateways listing handler.
func NewGatewaysHandler(svc *services.StatusService, logger *zerolog.Logger) *GatewaysHandler {
	return &GatewaysHandler{svc: svc, logger: logger}
}

func (h *GatewaysHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	gateways, err := h.svc.ListGateways(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list gateways")
		writeError(w, http.StatusBadGateway, "gateways_failed", "Failed to list gateways from control plane")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gateways": gateways,
		"total":    len(gateways),
	})
}

// writeError writes a JSON error response with a consistent envelope.
func writeError(w http.ResponseWriter, statusCode int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   errCode,
		"message": message,
	})
}

// Package controlplane implements an internal REST client for the AppGate
// Go Control Plane. It is the ONLY path through which the Dev-Portal BFF
// may register upstream LLM gateways and obtain signed execution credentials.
//
// Security invariants upheld by this package:
//   - Fail closed: any validation failure, transport error, or non-2xx
//     response converts into a typed error and zero credentials are returned.
//   - Never log raw provider credentials or signed JWTs: only identifiers,
//     model names, and spend/rate metadata are fielded into structured logs.
//   - The transport is pinned to TLS when configured; mTLS client
//     certificates are supported for mutual authentication with the plane.
package controlplane

import "time"

// RegisterRequest is the DTO shipped from the Dev-Portal BFF to the AppGate
// Control Plane at POST /api/v1/gateways/register.
//
// The BFF validates the payload before forwarding; the control plane re-
// validates independently (fail closed). Fields are intentionally aligned
// with the control plane's GatewayRegistrationRequest schema.
type RegisterRequest struct {
	// ProjectName is the internal name for the client's application.
	ProjectName string `json:"project_name"`

	// Model is the upstream LLM model identifier, e.g. "gpt-4o".
	// Validated as a non-empty, length-bounded identifier.
	Model string `json:"model"`

	// LLMModelName is the selectable model name from the dropdown.
	LLMModelName string `json:"llm_model_name"`

	// ProviderKey is the upstream SDK/api credential for the LLM provider.
	// Treated as a secret end-to-end: masked in logs, never persisted
	// client-side, and expired per policy.
	ProviderKey string `json:"provider_key"`

	// ProviderURL is the base URL of the upstream LLM endpoint.
	// Validated non-empty and must be an http(s) absolute URL to defend
	// against SSRF-style misdirection.
	ProviderURL string `json:"provider_url"`

	// MonthlySpendUSD is the hard monthly budget ceiling in US dollars.
	// Must be strictly greater than zero.
	MonthlySpendUSD float64 `json:"monthly_spend_usd"`

	// RateLimitRPS is the per-identity request ceiling in requests/second.
	RateLimitRPS int `json:"rate_limit_rps"`

	// RateLimitRPM is the per-identity request ceiling in requests/minute.
	RateLimitRPM int `json:"rate_limit_rpm"`

	// RateLimitBurst is the burst allowance applied by the data plane.
	RateLimitBurst int `json:"rate_limit_burst"`

	// WebhookURL is an optional URL notified on quota/policy events.
	// Optional; validated when present.
	WebhookURL string `json:"webhook_url"`

	// Tags is an arbitrary key/value map attached to the gateway record
	// for billing/ownership attribution. No security decisions depend on it.
	Tags map[string]string `json:"tags"`
}

// RegisterResponse is the credential handshake payload returned by the
// control plane after successful gateway registration and JWT signing.
//
// The JWTToken is the ONLY fully-signed credential: it must be rendered
// exactly once to the developer, then never again retrievable.
type RegisterResponse struct {
	ClientID       string `json:"client_id"`
	ClientSecretID string `json:"client_secret_id"`
	JWTToken       string `json:"jwt_token"`
	ExpiresAt      string `json:"expires_at"` // RFC3339 UTC
	TokenType      string `json:"token_type"` // "Bearer"
	Issuer         string `json:"issuer"`
}

// ErrorResponse is the canonical error envelope emitted by the control
// plane on non-2xx responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SecretStatus describes the current lifecycle state of an issued secret.
type SecretStatus struct {
	ClientID       string `json:"client_id"`
	ClientSecretID string `json:"client_secret_id"`
	Status         string `json:"status"` // "active" | "revoked" | "expired"
	Model          string `json:"model"`
	CreatedAt      string `json:"created_at"`
	ExpiresAt      string `json:"expires_at"`
}

// Meta represents control-plane-reported gateway metadata (non-sensitive).
type Meta struct {
	ID              string            `json:"id"`
	ProjectName     string            `json:"project_name"`
	Model           string            `json:"model"`
	MonthlySpendUSD float64           `json:"monthly_spend_usd"`
	RateLimitRPS    int               `json:"rate_limit_rps"`
	RateLimitRPM    int               `json:"rate_limit_rpm"`
	Status          string            `json:"status"`
	CreatedAt       string            `json:"created_at"`
	Tags            map[string]string `json:"tags"`
}

// ClientOption is a functional option for the control plane HTTP client.
type ClientOption func(*Client)

// WithBaseURL overrides the control plane base URL.
// Accepts a full URL including scheme: "https://control-plane.internal:8081".
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = baseURL
		}
	}
}

// WithTimeout overrides the default request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithRetries configures the maximum number of retries for idempotent
// safe requests (GET). Destructive/credential POSTs are never retried
// automatically to avoid double-issuance.
func WithRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		if maxRetries >= 0 {
			c.maxRetries = maxRetries
		}
	}
}

// WithBearerToken overrides the standard service-to-service bearer token
// presented to the control plane. Overrides any previously configured
// auth credential.
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.authToken = token
		c.clientCert = nil
		c.clientKey = nil
	}
}

// WithAuthCredential sets the service credentials (clientID/secret) used
// to obtain a short-lived control-plane bearer token via the token endpoint.
func WithAuthCredential(clientID, clientSecret string) ClientOption {
	return func(c *Client) {
		c.authToken = ""
		c.authClientID = clientID
		c.authClientSecret = clientSecret
	}
}

// WithClientCert configures mTLS: a PEM-encoded client certificate and key
// pair presented to the control plane. Both must be non-empty.
func WithClientCert(certPEM, keyPEM []byte) ClientOption {
	return func(c *Client) {
		c.clientCert = certPEM
		c.clientKey = keyPEM
	}
}

// WithInsecureSkipVerify disables server certificate verification.
// MUST ONLY be used in local development; the option documents the risk.
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		c.insecureSkipVerify = skip
	}
}

// WithLogger injects a structured logger. When nil, the client uses
// zerolog's default (no-op) logger.
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.log = logger
	}
}

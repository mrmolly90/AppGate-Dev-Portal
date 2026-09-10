package controlplane

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Defaults applied to every client unless overridden by options.
const (
	DefaultBaseURL         = "https://control-plane.internal:8081"
	DefaultTimeout         = 15 * time.Second
	DefaultMaxRetries      = 2
	RegisterPath           = "/api/v1/gateways/register"
	TokenPath              = "/oauth/token"
	SecretStatusPathPrefix = "/api/v1/secrets/"
)

// Logger is the minimal structured-logging surface the client requires.
// It is satisfied by *zerolog.Logger.
type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
}

// Client is a fail-closed HTTP client for the AppGate Control Plane.
// Safe for concurrent use.
type Client struct {
	baseURL            string
	httpClient         *http.Client
	maxRetries         int
	log                Logger
	authToken          string
	authClientID       string
	authClientSecret   string
	clientCert         []byte
	clientKey          []byte
	insecureSkipVerify bool
}

// New creates a control plane client with secure defaults: TLS-required,
// bounded timeouts, retries for idempotent reads only, and no credential
// material in structured logs.
func New(opts ...ClientOption) (*Client, error) {
	c := &Client{
		baseURL:    DefaultBaseURL,
		maxRetries: DefaultMaxRetries,
		log:        nil,
	}
	c.httpClient = &http.Client{Timeout: DefaultTimeout}
	for _, opt := range opts {
		opt(c)
	}

	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	if len(c.clientCert) > 0 && len(c.clientKey) > 0 {
		cert, err := tls.X509KeyPair(c.clientCert, c.clientKey)
		if err != nil {
			return nil, fmt.Errorf("controlplane: invalid mTLS client certificate: %w", err)
		}
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: c.insecureSkipVerify,
		}
	} else if c.insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	}

	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("controlplane: invalid base URL %q: scheme must be https (or http in dev)", c.baseURL)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("controlplane: invalid base URL %q: missing host", c.baseURL)
	}
	c.httpClient.Transport = transport

	if c.log == nil {
		discard := zerolog.New(io.Discard)
		c.log = &discard
	}
	return c, nil
}

// logf emits a structured log event with key/value fields.
// Callers MUST NOT pass secrets (provider keys, JWTs) as values.
func (c *Client) logf(level string, msg string, kv ...any) {
	if c.log == nil {
		return
	}
	var ev *zerolog.Event
	switch level {
	case "error":
		ev = c.log.Error()
	case "warn":
		ev = c.log.Warn()
	case "debug":
		ev = c.log.Debug()
	default:
		ev = c.log.Info()
	}
	if len(kv) > 0 {
		fields := make(map[string]any, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			key, ok := kv[i].(string)
			if !ok {
				continue
			}
			fields[key] = kv[i+1]
		}
		ev.Fields(fields)
	}
	ev.Msg(msg)
}

// Register submits a gateway registration to the AppGate Control Plane and
// performs the full credential handshake. Fail closed: returns typed error
// and zero credentials on any validation or transport failure.
func (c *Client) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if req == nil {
		return nil, errors.New("controlplane: nil registration request")
	}
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("controlplane: marshal request: %w", err)
	}
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(c.baseURL, "/") + RegisterPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("controlplane: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	c.logf("info", "control plane gateway registration",
		"project", req.ProjectName,
		"model", req.Model,
		"spend_usd", req.MonthlySpendUSD,
		"rps", req.RateLimitRPS)

	// No automatic retries on POST: credential issuance is non-idempotent.
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("controlplane: register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr ErrorResponse
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&apiErr)
		return nil, fmt.Errorf("controlplane: registration failed (http %d): %s: %s",
			resp.StatusCode, apiErr.Error, apiErr.Message)
	}

	var out RegisterResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("controlplane: decode register response: %w", err)
	}
	if err := validateRegisterResponse(&out); err != nil {
		return nil, err
	}

	c.logf("info", "control plane registration completed",
		"client_id", out.ClientID,
		"client_secret_id", out.ClientSecretID,
		"token_type", out.TokenType,
		"expires_at", out.ExpiresAt)

	return &out, nil
}

// SecretStatus fetches the lifecycle status of an issued secret.
func (c *Client) SecretStatus(ctx context.Context, clientID string) (*SecretStatus, error) {
	if clientID == "" {
		return nil, errors.New("controlplane: empty client ID")
	}
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(c.baseURL, "/") + SecretStatusPathPrefix + url.PathEscape(clientID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("controlplane: build request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.doWithRetries(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr ErrorResponse
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&apiErr)
		return nil, fmt.Errorf("controlplane: secret status failed (http %d): %s: %s",
			resp.StatusCode, apiErr.Error, apiErr.Message)
	}
	var out SecretStatus
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("controlplane: decode secret status: %w", err)
	}
	return &out, nil
}

// ListGateways fetches all registered gateways from the control plane.
func (c *Client) ListGateways(ctx context.Context) ([]Meta, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(c.baseURL, "/") + "/api/v1/gateways"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("controlplane: build request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.doWithRetries(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr ErrorResponse
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&apiErr)
		return nil, fmt.Errorf("controlplane: list gateways failed (http %d): %s: %s",
			resp.StatusCode, apiErr.Error, apiErr.Message)
	}
	var listResp struct {
		Gateways []Meta `json:"gateways"`
		Total    int    `json:"total"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("controlplane: decode gateways list: %w", err)
	}
	return listResp.Gateways, nil
}

// Health performs a GET on the control plane health endpoint.
func (c *Client) Health(ctx context.Context) error {
	endpoint := strings.TrimRight(c.baseURL, "/") + "/health"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("controlplane: build request: %w", err)
	}
	resp, err := c.doWithRetries(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("controlplane: health check failed (http %d)", resp.StatusCode)
	}
	return nil
}

// doWithRetries performs an HTTP request with bounded retries on transient
// failures (connection refused, 5xx, 429). Safe for idempotent reads.
func (c *Client) doWithRetries(req *http.Request) (*http.Response, error) {
	var lastErr error
	attempts := c.maxRetries + 1
	for i := 0; i < attempts; i++ {
		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			_ = resp.Body.Close()
		}
		if i < attempts-1 {
			backoff := time.Duration(1<<i) * 100 * time.Millisecond
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff):
			}
		}
	}
	return nil, fmt.Errorf("controlplane: request failed after retries: %w", lastErr)
}

// ensureToken returns a valid bearer token for the control plane.
func (c *Client) ensureToken(ctx context.Context) (string, error) {
	if c.authToken != "" {
		return c.authToken, nil
	}
	if c.authClientID == "" || c.authClientSecret == "" {
		return "", nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.authClientID)
	form.Set("client_secret", c.authClientSecret)
	form.Set("audience", "appgate-control-plane")
	endpoint := strings.TrimRight(c.baseURL, "/") + TokenPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("controlplane: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("controlplane: token request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("controlplane: token request failed (http %d)", resp.StatusCode)
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("controlplane: decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("controlplane: token response missing access_token")
	}
	return tokenResp.AccessToken, nil
}

// validateRegisterRequest enforces local sanity constraints before any
// network I/O. Fail closed: any violation aborts the request.
func validateRegisterRequest(req *RegisterRequest) error {
	if req == nil {
		return errors.New("controlplane: nil registration request")
	}
	if strings.TrimSpace(req.ProjectName) == "" {
		return errors.New("controlplane: project_name is required")
	}
	if len(req.ProjectName) > 128 {
		return errors.New("controlplane: project_name exceeds 128 characters")
	}
	if strings.TrimSpace(req.Model) == "" {
		return errors.New("controlplane: model is required")
	}
	if len(req.Model) > 128 {
		return errors.New("controlplane: model exceeds 128 characters")
	}
	if strings.TrimSpace(req.ProviderKey) == "" {
		return errors.New("controlplane: provider key is required")
	}
	if len(req.ProviderKey) < 8 {
		return errors.New("controlplane: provider key must be at least 8 characters")
	}
	if strings.TrimSpace(req.ProviderURL) == "" {
		return errors.New("controlplane: provider URL is required")
	}
	pu, err := url.Parse(req.ProviderURL)
	if err != nil || (pu.Scheme != "https" && pu.Scheme != "http") || pu.Host == "" {
		return errors.New("controlplane: provider URL must be an absolute https URL")
	}
	// SSRF defense: upstream URL must use HTTPS.
	if pu.Scheme == "http" {
		return errors.New("controlplane: provider URL must use https, not http")
	}
	if req.MonthlySpendUSD <= 0 {
		return errors.New("controlplane: monthly spend must be greater than zero")
	}
	return nil
}

// validateRegisterResponse verifies the credential handshake is complete
// before any value is returned to the BFF. Fail closed.
func validateRegisterResponse(resp *RegisterResponse) error {
	if resp == nil {
		return errors.New("controlplane: nil register response")
	}
	if strings.TrimSpace(resp.ClientID) == "" {
		return errors.New("controlplane: response missing client_id")
	}
	if strings.TrimSpace(resp.ClientSecretID) == "" {
		return errors.New("controlplane: response missing client_secret_id")
	}
	if strings.TrimSpace(resp.JWTToken) == "" {
		return errors.New("controlplane: response missing jwt_token")
	}
	if resp.TokenType != "" && !strings.EqualFold(resp.TokenType, "bearer") {
		return errors.New("controlplane: unexpected token type in response")
	}
	return nil
}

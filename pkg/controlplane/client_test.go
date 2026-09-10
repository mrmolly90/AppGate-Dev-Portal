package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *RegisterRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name:    "missing model",
			req:     &RegisterRequest{ProjectName: "my-app", ProviderKey: "sk-abcdefgh", ProviderURL: "https://api.example.com", MonthlySpendUSD: 100},
			wantErr: true,
		},
		{
			name:    "missing provider key",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderURL: "https://api.example.com", MonthlySpendUSD: 100},
			wantErr: true,
		},
		{
			name:    "short provider key",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderKey: "short", ProviderURL: "https://api.example.com", MonthlySpendUSD: 100},
			wantErr: true,
		},
		{
			name:    "missing provider URL",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderKey: "sk-abcdefgh", MonthlySpendUSD: 100},
			wantErr: true,
		},
		{
			name:    "HTTP provider URL (should fail SSRF check)",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderKey: "sk-abcdefgh", ProviderURL: "http://api.example.com", MonthlySpendUSD: 100},
			wantErr: true,
		},
		{
			name:    "zero spend",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderKey: "sk-abcdefgh", ProviderURL: "https://api.example.com", MonthlySpendUSD: 0},
			wantErr: true,
		},
		{
			name:    "valid request",
			req:     &RegisterRequest{ProjectName: "my-app", Model: "gpt-4o", ProviderKey: "sk-abcdefgh", ProviderURL: "https://api.example.com", MonthlySpendUSD: 500},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRegisterResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    *RegisterResponse
		wantErr bool
	}{
		{"nil response", nil, true},
		{"missing client_id", &RegisterResponse{ClientSecretID: "sec-123", JWTToken: "eyJhbGci"}, true},
		{"missing secret_id", &RegisterResponse{ClientID: "gw-abc", JWTToken: "eyJhbGci"}, true},
		{"missing jwt", &RegisterResponse{ClientID: "gw-abc", ClientSecretID: "sec-123"}, true},
		{"wrong token type", &RegisterResponse{ClientID: "gw-abc", ClientSecretID: "sec-123", JWTToken: "eyJhbGci", TokenType: "Basic"}, true},
		{"valid minimal", &RegisterResponse{ClientID: "gw-abc", ClientSecretID: "sec-123", JWTToken: "eyJhbGci"}, false},
		{"valid full", &RegisterResponse{ClientID: "gw-abc", ClientSecretID: "sec-123", JWTToken: "eyJhbGci", TokenType: "Bearer"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterResponse(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterResponse() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientNew_InvalidURL(t *testing.T) {
	_, err := New(WithBaseURL("not-a-url"))
	if err == nil {
		t.Error("New() with invalid URL should fail")
	}
}

func TestClientNew_EmptyHost(t *testing.T) {
	_, err := New(WithBaseURL("https://"))
	if err == nil {
		t.Error("New() with empty host should fail")
	}
}

func TestClientRegister_NilRequest(t *testing.T) {
	c, err := New(WithBaseURL("https://control-plane.test:8081"))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, err = c.Register(context.Background(), nil)
	if err == nil {
		t.Error("Register(nil) should fail")
	}
}

func TestClientSecretStatus_EmptyID(t *testing.T) {
	c, err := New(WithBaseURL("https://control-plane.test:8081"))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, err = c.SecretStatus(context.Background(), "")
	if err == nil {
		t.Error("SecretStatus('') should fail")
	}
}

func TestClientNetworkError(t *testing.T) {
	// Client pointing to a non-routable address should fail fast
	c, err := New(WithBaseURL("https://192.0.2.1:8081"), WithTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, err = c.Register(context.Background(), &RegisterRequest{
		ProjectName:     "test",
		Model:           "test",
		ProviderKey:     "sk-abcdefgh",
		ProviderURL:     "https://api.example.com",
		MonthlySpendUSD: 100,
	})
	if err == nil {
		t.Error("Register() to unreachable address should fail")
	}
}

func TestClientOptions(t *testing.T) {
	// Verify functional options are applied
	client, err := New(
		WithBaseURL("http://localhost:8081"),
		WithTimeout(5*time.Second),
		WithRetries(3),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if client.baseURL != "http://localhost:8081" {
		t.Errorf("baseURL = %q; want http://localhost:8081", client.baseURL)
	}
	if client.maxRetries != 3 {
		t.Errorf("maxRetries = %d; want 3", client.maxRetries)
	}
}

func TestClientBearerToken(t *testing.T) {
	client, err := New(
		WithBaseURL("http://localhost:8081"),
		WithBearerToken("test-token"),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if client.authToken != "test-token" {
		t.Errorf("authToken = %q; want test-token", client.authToken)
	}
	// Bearer token should clear mTLS certs
	if client.clientCert != nil || client.clientKey != nil {
		t.Error("Bearer token option should clear mTLS certs")
	}
}

func TestClientmTLS(t *testing.T) {
	// mTLS with invalid PEM should fail
	_, err := New(
		WithBaseURL("http://localhost:8081"),
		WithClientCert([]byte("invalid-cert"), []byte("invalid-key")),
	)
	if err == nil {
		t.Error("Client with invalid mTLS PEM should fail")
	}
}

func TestClient_IntegrationViaTestServer(t *testing.T) {
	// Start a test HTTP server that mimics the control plane
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == RegisterPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"client_id": "gw-test-001",
				"client_secret_id": "sec-test-001",
				"jwt_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-token",
				"expires_at": "2025-01-01T00:00:00Z",
				"token_type": "Bearer",
				"issuer": "appgate"
			}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := New(WithBaseURL(server.URL), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := c.Register(context.Background(), &RegisterRequest{
		ProjectName:     "test-app",
		Model:           "gpt-4o",
		ProviderKey:     "sk-abcdefgh",
		ProviderURL:     "https://api.openai.com",
		MonthlySpendUSD: 500,
		RateLimitRPS:    100,
		RateLimitBurst:  200,
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if resp.ClientID != "gw-test-001" {
		t.Errorf("ClientID = %q; want gw-test-001", resp.ClientID)
	}
	if resp.JWTToken == "" {
		t.Error("JWTToken should not be empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q; want Bearer", resp.TokenType)
	}

	// Test health
	if err := c.Health(context.Background()); err != nil {
		t.Errorf("Health() failed: %v", err)
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_request","message":"Provider key too short"}`))
	}))
	defer server.Close()

	c, err := New(WithBaseURL(server.URL), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = c.Register(context.Background(), &RegisterRequest{
		Model:           "gpt-4o",
		ProviderKey:     "sk-abcdefgh",
		ProviderURL:     "https://api.openai.com",
		MonthlySpendUSD: 500,
	})
	if err == nil {
		t.Fatal("Register() should have failed with 400")
	}
	t.Logf("Got expected error: %v", err)
}

func TestClient_ListGateways(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/gateways" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"gateways": [
					{
						"id": "gw-test-001",
						"model": "gpt-4o",
						"monthly_spend_usd": 500,
						"rate_limit_rps": 100,
						"status": "active",
						"created_at": "2025-01-01T00:00:00Z",
						"tags": {"team": "payments"}
					}
				],
				"total": 1
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := New(WithBaseURL(server.URL), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	gateways, err := c.ListGateways(context.Background())
	if err != nil {
		t.Fatalf("ListGateways() failed: %v", err)
	}
	if len(gateways) != 1 {
		t.Fatalf("got %d gateways; want 1", len(gateways))
	}
	if gateways[0].ID != "gw-test-001" {
		t.Errorf("gateway ID = %q; want gw-test-001", gateways[0].ID)
	}
	if gateways[0].Model != "gpt-4o" {
		t.Errorf("model = %q; want gpt-4o", gateways[0].Model)
	}
	if gateways[0].Status != "active" {
		t.Errorf("status = %q; want active", gateways[0].Status)
	}
}

func TestClient_ListGateways_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server_error","message":"Internal error"}`))
	}))
	defer server.Close()

	c, err := New(WithBaseURL(server.URL), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = c.ListGateways(context.Background())
	if err == nil {
		t.Fatal("ListGateways() should have failed with 500")
	}
	t.Logf("Got expected error: %v", err)
}

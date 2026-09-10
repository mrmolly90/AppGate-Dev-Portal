package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/appgate/dev-portal/apps/api/services"
	"github.com/appgate/dev-portal/pkg/controlplane"
	"github.com/rs/zerolog"
)

func testLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

// fakeControlPlane spins up an httptest server that mimics the AppGate
// control plane register endpoint.
func fakeControlPlane(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == controlplane.RegisterPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"client_id": "gw-cp-001",
				"client_secret_id": "sec-cp-001",
				"jwt_token": "eyJhbGciOiJSUzI1NiJ9.fake.jwt",
				"expires_at": "2030-01-01T00:00:00Z",
				"token_type": "Bearer",
				"issuer": "appgate"
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not_found","message":"unknown route"}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRegisterHandler_Success(t *testing.T) {
	cp := fakeControlPlane(t)
	client, err := controlplane.New(
		controlplane.WithBaseURL(cp.URL),
		controlplane.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create control plane client: %v", err)
	}
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	body := `{
		"project_name": "test-app",
		"model": "gpt-4o",
		"provider_key": "sk-super-secret-key-1",
		"provider_url": "https://api.openai.com",
		"monthly_spend_usd": 500,
		"rate_limit_rps": 100,
		"rate_limit_burst": 200
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/tenant/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Status = %d; want 201. Body: %s", w.Code, w.Body.String())
	}

	var resp TenantRegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.ClientID != "gw-cp-001" {
		t.Errorf("ClientID = %q; want gw-cp-001", resp.ClientID)
	}
	if resp.JWTToken == "" {
		t.Error("JWTToken should not be empty")
	}
	if resp.ClientSecretID != "sec-cp-001" {
		t.Errorf("ClientSecretID = %q; want sec-cp-001", resp.ClientSecretID)
	}

	// No-store header must be set on credential responses
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Errorf("Cache-Control = %q; want it to contain no-store", cc)
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	cp := fakeControlPlane(t)
	client, _ := controlplane.New(controlplane.WithBaseURL(cp.URL))
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	// Missing provider_key
	body := `{
		"project_name": "test-app",
		"model": "gpt-4o",
		"provider_url": "https://api.openai.com",
		"monthly_spend_usd": 500
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/tenant/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d; want 400", w.Code)
	}
}

func TestRegisterHandler_ShortProviderKey(t *testing.T) {
	cp := fakeControlPlane(t)
	client, _ := controlplane.New(controlplane.WithBaseURL(cp.URL))
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	body := `{
		"project_name": "test-app",
		"model": "gpt-4o",
		"provider_key": "short",
		"provider_url": "https://api.openai.com",
		"monthly_spend_usd": 500
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/tenant/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d; want 400", w.Code)
	}
}

func TestRegisterHandler_HTTPProviderURLRejected(t *testing.T) {
	cp := fakeControlPlane(t)
	client, _ := controlplane.New(controlplane.WithBaseURL(cp.URL))
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	body := `{
		"project_name": "test-app",
		"model": "gpt-4o",
		"provider_key": "sk-super-secret-key-1",
		"provider_url": "http://api.openai.com",
		"monthly_spend_usd": 500
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/tenant/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d; want 400 (SSRF defense)", w.Code)
	}
}

func TestRegisterHandler_WrongMethod(t *testing.T) {
	cp := fakeControlPlane(t)
	client, _ := controlplane.New(controlplane.WithBaseURL(cp.URL))
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/tenant/register", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status = %d; want 405", w.Code)
	}
}

func TestRegisterHandler_ControlPlaneDown(t *testing.T) {
	// Point at a closed server -> BFF must return 502 (fail closed)
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := closedServer.URL
	closedServer.Close()

	client, err := controlplane.New(
		controlplane.WithBaseURL(deadURL),
		controlplane.WithTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	svc := services.NewRegistrationService(client)
	h := NewRegisterHandler(svc, testLogger())

	body := `{
		"project_name": "test-app",
		"model": "gpt-4o",
		"provider_key": "sk-super-secret-key-1",
		"provider_url": "https://api.openai.com",
		"monthly_spend_usd": 500
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/tenant/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("Status = %d; want 502 (fail closed when control plane is down)", w.Code)
	}
}

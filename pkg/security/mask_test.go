package security

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"sk-proj-ABC123xyz", "sk****yz"},
		{"sk-proj-a1b2c3d4", "sk****d4"},
		{"a", "****"},
	}
	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaskURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"https://example.com/path", "https://example.com/path"},
		{"https://token:secret@example.com", "https://****@****example.com"},
		{"http://user@host.com", "http://****@****host.com"},
	}
	for _, tt := range tests {
		got := MaskURL(tt.input)
		if got != tt.expected {
			t.Errorf("MaskURL(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaskAll(t *testing.T) {
	if got := MaskAll(); got != "****" {
		t.Errorf("MaskAll() = %q; want %q", got, "****")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long string", 9, "this is a..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := Truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q; want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

// TestParseTokenUnverified ensures we can extract claims without verification.
func TestParseTokenUnverified(t *testing.T) {
	// Build a minimal JWT (no signature verification needed)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   "gw-test-123",
		Issuer:    "appgate",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})
	// Sign with a real generated key for structural validity
	tokenStr, err := token.SignedString(testPrivateKey(t))
	if err != nil {
		t.Fatalf("Failed to sign test token: %v", err)
	}

	claims, err := ParseTokenUnverified(tokenStr)
	if err != nil {
		t.Fatalf("ParseTokenUnverified() error: %v", err)
	}
	if claims == nil || claims.Subject != "gw-test-123" {
		t.Errorf("ParseTokenUnverified() returned wrong Subject: got %v", claims)
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		claims *Claims
		want   bool
	}{
		{"nil claims", nil, false},
		{"expired", &Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			},
		}, true},
		{"not expired", &Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			},
		}, false},
		{"no expiry", &Claims{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.claims.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestValidateJTI(t *testing.T) {
	if ValidateJTI(nil) {
		t.Error("ValidateJTI(nil) should be false")
	}
	if ValidateJTI(&Claims{}) {
		t.Error("ValidateJTI(empty) should be false")
	}
	if !ValidateJTI(&Claims{RegisteredClaims: jwt.RegisteredClaims{ID: "abc-123"}}) {
		t.Error("ValidateJTI(valid) should be true")
	}
}

// testPrivateKey generates an ephemeral RSA key for test-only token signing.
func testPrivateKey(t *testing.T) interface{} {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}
	return key
}

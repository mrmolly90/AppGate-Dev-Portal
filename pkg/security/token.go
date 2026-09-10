package security

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned when a JWT fails structural validation.
var ErrInvalidToken = errors.New("security: invalid JWT token")

// ErrTokenExpired is returned when a JWT's expiration has passed.
var ErrTokenExpired = errors.New("security: JWT token expired")

// Claims is the minimal claim set the data plane (Rust gateway) validates.
// It mirrors rust-gateway/src/jwt.rs Claims and is the contract for any
// token issued by the control plane.
type Claims struct {
	jwt.RegisteredClaims
	Roles    []string `json:"roles,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
}

// ParseToken parses and structurally validates a JWT without verifying its
// signature. Use this only for introspection (e.g. masking, display).
// For authorization decisions, always verify the signature with the
// control plane's public key.
func ParseToken(tokenString string) (*Claims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256", "ES256", "EdDSA"}))
	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Keyfunc returns an error marker; the caller MUST NOT use the
		// parsed token for authorization without verifying the signature.
		return nil, errors.New("signature verification required")
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseTokenUnverified parses a token and returns its payload claims
// without any validation. Intended ONLY for display/masking of JWTs that
// were already received from the control plane over a trusted channel.
func ParseTokenUnverified(tokenString string) (*Claims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}
	claims := &Claims{}
	parser := jwt.NewParser()
	if _, _, err := parser.ParseUnverified(tokenString, claims); err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// IsExpired reports whether the claims' expiration has passed.
func (c *Claims) IsExpired() bool {
	if c == nil || c.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(c.ExpiresAt.Time)
}

// ExpirationTime returns the expiry as a time.Time (zero if unset).
func (c *Claims) ExpirationTime() time.Time {
	if c == nil || c.ExpiresAt == nil {
		return time.Time{}
	}
	return c.ExpiresAt.Time
}

// IsValidUUID reports whether s is a well-formed v4 UUID.
// Used to validate client_id / secret_id values before they are trusted
// for lookup or persistence.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// ValidateJTI ensures a JWT ID is present and non-empty, which the
// control plane always embeds. Tokens lacking a jti are treated as
// malformed for replay-safety bookkeeping.
func ValidateJTI(c *Claims) bool {
	return c != nil && strings.TrimSpace(c.ID) != ""
}

// IsLikelySecret reports whether a string resembles a high-entropy
// credential. Short strings and whitespace are not secrets.
func IsLikelySecret(s string) bool {
	if len(s) < 8 {
		return false
	}
	upper, lower, digit, special := 0, 0, 0, 0
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			upper++
		case r >= 'a' && r <= 'z':
			lower++
		case r >= '0' && r <= '9':
			digit++
		default:
			special++
		}
	}
	return upper+lower > 0 && digit > 0 || special >= 2
}

package jwt

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Signer handles JWT token signing operations.
type Signer struct {
	privateKey interface{}
	method     jwt.SigningMethod
	issuer     string
	audience   string
	expiry     time.Duration
}

// SignerConfig holds configuration for the JWT signer.
type SignerConfig struct {
	PrivateKeyPath string
	PrivateKeyPEM  string
	Issuer         string
	Audience       string
	Expiry         time.Duration
	SigningMethod  string // "RS256" or "ES256"
}

// NewSigner creates a new JWT signer from the given configuration.
// Supports both file-based and inline PEM-encoded private keys.
func NewSigner(cfg SignerConfig) (*Signer, error) {
	var pemBlock []byte

	switch {
	case cfg.PrivateKeyPEM != "":
		pemBlock = []byte(cfg.PrivateKeyPEM)
	case cfg.PrivateKeyPath != "":
		data, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}
		pemBlock = data
	default:
		return nil, errors.New("no private key provided: set CP_JWT_PRIVATE_KEY_PATH or CP_JWT_PRIVATE_KEY_PEM")
	}

	block, _ := pem.Decode(pemBlock)
	if block == nil {
		return nil, errors.New("failed to decode PEM block from private key")
	}

	var signingMethod jwt.SigningMethod
	var privateKey interface{}
	var err error

	switch cfg.SigningMethod {
	case "RS256":
		privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
			}
		}
		if _, ok := privateKey.(*rsa.PrivateKey); !ok {
			return nil, errors.New("key is not an RSA private key")
		}
		signingMethod = jwt.SigningMethodRS256
	case "ES256":
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse EC private key: %w", err)
			}
		}
		if _, ok := privateKey.(*ecdsa.PrivateKey); !ok {
			return nil, errors.New("key is not an ECDSA private key")
		}
		signingMethod = jwt.SigningMethodES256
	default:
		return nil, fmt.Errorf("unsupported signing method: %s (supported: RS256, ES256)", cfg.SigningMethod)
	}

	issuer := cfg.Issuer
	if issuer == "" {
		issuer = "appgate"
	}
	audience := cfg.Audience
	if audience == "" {
		audience = "appgate-api"
	}
	expiry := cfg.Expiry
	if expiry <= 0 {
		expiry = 1 * time.Hour
	}

	return &Signer{
		privateKey: privateKey,
		method:     signingMethod,
		issuer:     issuer,
		audience:   audience,
		expiry:     expiry,
	}, nil
}

// GatewayClaims represents the claims embedded in a gateway JWT.
// These match the Claims struct in the Rust gateway's jwt.rs validator.
type GatewayClaims struct {
	jwt.RegisteredClaims
	Roles    []string `json:"roles,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
}

// SignGatewayToken creates a signed JWT for a gateway client.
// It embeds the client_id, roles, and scope matching what the Rust gateway
// validator expects (sub, iss, aud, exp, iat, nbf, jti, roles, scope).
func (s *Signer) SignGatewayToken(clientID string, roles []string, scope string) (string, error) {
	now := time.Now().UTC()
	jti := uuid.New().String()

	if roles == nil {
		roles = []string{}
	}
	if scope == "" {
		scope = "gateway:proxy"
	}

	claims := GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   clientID,
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)), // 30s clock skew leeway
			ID:        jti,
		},
		Roles:    roles,
		Scope:    scope,
		ClientID: clientID,
	}

	token := jwt.NewWithClaims(s.method, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signed, nil
}

// Method returns the signing method in use.
func (s *Signer) Method() string {
	return s.method.Alg()
}

// Issuer returns the configured issuer.
func (s *Signer) Issuer() string {
	return s.issuer
}

// Audience returns the configured audience.
func (s *Signer) Audience() string {
	return s.audience
}

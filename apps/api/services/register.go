// Package services contains the orchestration and business logic of the
// Dev-Portal BFF: wiring the HTTP handlers to the control plane client
// and enforcing fail-closed end-to-end semantics.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/appgate/dev-portal/pkg/controlplane"
)

// ErrProviderKeyDisclosed is returned when a registration request attempts
// to reveal a provider key that does not match a previously stored digest.
// This never happens through the public API (the BFF never persists keys)
// and is treated as a security violation.
var ErrProviderKeyDisclosed = errors.New("services: provider key disclosure detected")

// ErrRegistrationRejected is returned when the control plane rejects the
// registration for policy reasons.
var ErrRegistrationRejected = errors.New("services: registration rejected by control plane")

// RegistrationService orchestrates the registration flow.
type RegistrationService struct {
	cp *controlplane.Client
}

// NewRegistrationService wires the control plane client into the service.
func NewRegistrationService(cp *controlplane.Client) *RegistrationService {
	return &RegistrationService{cp: cp}
}

// Register forwards a validated request to the control plane and returns
// the credential handshake. It never logs the provider key or the JWT.
func (s *RegistrationService) Register(ctx context.Context, req *controlplane.RegisterRequest) (*controlplane.RegisterResponse, error) {
	if req == nil {
		return nil, errors.New("services: nil registration request")
	}
	if s.cp == nil {
		return nil, errors.New("services: control plane client not configured")
	}
	resp, err := s.cp.Register(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistrationRejected, err)
	}
	return resp, nil
}

// KeyDigest returns a SHA-256 digest of a provider key. Used to detect
// accidental reuse disclosure without retaining the raw key.
func KeyDigest(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// MaskProviderKey returns a masked representation for UI echo purposes.
// Never send the raw key back to the client.
func MaskProviderKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[:2] + "****" + key[len(key)-2:]
}

// ExpiryFromToken parses the "expires_at" RFC3339 string returned by the
// control plane into a time.Time for client display.
func ExpiryFromToken(expiresAt string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("services: invalid expires_at: %w", err)
	}
	return t, nil
}

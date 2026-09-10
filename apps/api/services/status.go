package services

import (
	"context"
	"errors"

	"github.com/appgate/dev-portal/pkg/controlplane"
)

// StatusService provides secret lifecycle queries against the control plane.
type StatusService struct {
	cp *controlplane.Client
}

// NewStatusService creates a status query service.
func NewStatusService(cp *controlplane.Client) *StatusService {
	return &StatusService{cp: cp}
}

// Status fetches the current lifecycle status of an issued secret.
func (s *StatusService) Status(ctx context.Context, clientID string) (*controlplane.SecretStatus, error) {
	if clientID == "" {
		return nil, errors.New("services: empty client ID")
	}
	if s.cp == nil {
		return nil, errors.New("services: control plane client not configured")
	}
	return s.cp.SecretStatus(ctx, clientID)
}

// ListGateways fetches all registered gateways from the control plane.
func (s *StatusService) ListGateways(ctx context.Context) ([]controlplane.Meta, error) {
	if s.cp == nil {
		return nil, errors.New("services: control plane client not configured")
	}
	return s.cp.ListGateways(ctx)
}

// Health reports whether the control plane is reachable and healthy.
func (s *StatusService) Health(ctx context.Context) error {
	if s.cp == nil {
		return errors.New("services: control plane client not configured")
	}
	return s.cp.Health(ctx)
}

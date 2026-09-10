package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// GatewayStore provides persistence for gateway records.
// In production, this would be backed by a database with encryption at rest.
type GatewayStore struct {
	path    string
	mu      sync.RWMutex
	records map[string]*GatewayRecord // keyed by ClientID
}

// NewGatewayStore creates a file-backed gateway store.
func NewGatewayStore(path string) (*GatewayStore, error) {
	s := &GatewayStore{
		path:    path,
		records: make(map[string]*GatewayRecord),
	}

	// Ensure the parent directory exists so persistence always succeeds
	if path != "" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}

	// Try to load existing data
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // Fresh store
		}
		return nil, err
	}

	var records []GatewayRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	for i := range records {
		s.records[records[i].ClientID] = &records[i]
	}

	return s, nil
}

// NewInMemoryStore creates an ephemeral in-memory store (for development).
func NewInMemoryStore() *GatewayStore {
	return &GatewayStore{
		path:    "",
		records: make(map[string]*GatewayRecord),
	}
}

// Save persists a gateway record (creates or updates).
func (s *GatewayStore) Save(record GatewayRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[record.ClientID] = &record

	if s.path != "" {
		return s.flush()
	}
	return nil
}

// Get retrieves a gateway record by client ID.
func (s *GatewayStore) Get(clientID string) (*GatewayRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[clientID]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid race conditions
	rec := *r
	return &rec, true
}

// List returns all gateway records (without sensitive fields).
func (s *GatewayStore) List() []GatewayRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]GatewayRecord, 0, len(s.records))
	for _, r := range s.records {
		result = append(result, *r)
	}
	return result
}

// Delete removes a gateway record by client ID.
func (s *GatewayStore) Delete(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, clientID)

	if s.path != "" {
		return s.flush()
	}
	return nil
}

// flush writes all records to disk.
func (s *GatewayStore) flush() error {
	records := make([]GatewayRecord, 0, len(s.records))
	for _, r := range s.records {
		records = append(records, *r)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0600)
}

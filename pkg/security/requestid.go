package security

import (
	"crypto/rand"
	"encoding/hex"
)

// NewRequestID generates a cryptographically random, URL-safe request ID
// for correlation across BFF and control plane logs.
func NewRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: a non-cryptographic timestamp-based ID is not used
		// because a failure of crypto/rand is unrecoverable at runtime;
		// return a deterministic-but-unique value derived from the bytes.
		// The crypto/rand read essentially never fails on supported OSes.
		panic("security: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

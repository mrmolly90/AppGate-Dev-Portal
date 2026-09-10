// Package security provides zero-trust utilities used across the
// Dev-Portal BFF: credential masking, JWT parsing/validation, and
// payload normalization. Nothing in this package ever logs secret
// material.
package security

import (
	"strings"
)

// maskChar is the replacement character for masked secrets.
const maskChar = "*"

// MaskSecret redacts all but the first and last two characters of a
// secret. Secrets shorter than 5 characters are fully redacted.
//
// Examples:
//
//	MaskSecret("sk-proj-ABC123")  -> "sk****23"
//	MaskSecret("abc")             -> "****"
func MaskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:2] + "****" + secret[len(secret)-2:]
}

// MaskURL redacts the userinfo portion of a URL, e.g.
//
//	"https://token:secret@example.com" -> "https://****:****@example.com"
func MaskURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	idx := strings.Index(rawURL, "://")
	if idx < 0 {
		return rawURL
	}
	scheme := rawURL[:idx+3]
	rest := rawURL[idx+3:]
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return rawURL
	}
	return scheme + "****@****" + rest[at+1:]
}

// MaskAll returns a fixed redaction string. Use for values that must
// never appear in logs in any form (e.g. full prompts, upstream keys).
func MaskAll() string {
	return "****"
}

// Truncate limits a string to maxLen runes, appending an ellipsis.
// Used to bound log fields on descriptions (never on secrets).
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

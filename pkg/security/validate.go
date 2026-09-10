package security

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Security-relevant validation helpers. All functions here FAIL CLOSED:
// an invalid or unknown input yields an error, never a permissive default.

// ErrUnsafeURL indicates a URL failed SSRF/allowlist validation.
var ErrUnsafeURL = errors.New("security: unsafe URL")

// IsPrivateHost reports whether a hostname resolves to a private,
// loopback, or link-local address. It performs a DNS lookup, so callers
// must be prepared for network latency and resolve failures (treated as
// unsafe — fail closed).
func IsPrivateHost(host string) bool {
	ips, err := net.LookupHost(host)
	if err != nil {
		// Fail closed: cannot resolve, treat as unsafe.
		return true
	}
	for _, ip := range ips {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			continue
		}
		if parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast() ||
			parsed.IsLinkLocalMulticast() || parsed.IsUnspecified() {
			return true
		}
	}
	return false
}

// ValidateOutboundURL enforces the AppGate SSRF policy for any outbound
// URL supplied by a client (e.g. webhook URLs, provider URLs when not
// allowlisted server-side):
//
//   - must be absolute http(s);
//   - must not be a private/loopback/link-local host (fail closed on
//     DNS resolution errors);
//   - must not contain userinfo (cannot smuggle credentials);
//   - must not use a non-standard port.
func ValidateOutboundURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrUnsafeURL
	}
	if u.Scheme != "https" {
		return ErrUnsafeURL
	}
	if u.User != nil {
		return ErrUnsafeURL
	}
	if u.Host == "" {
		return ErrUnsafeURL
	}

	host := u.Hostname()
	port := u.Port()
	if port != "" && port != "443" && port != "8443" {
		// Restrict to standard/approved ports to reduce attack surface.
		return ErrUnsafeURL
	}

	if IsPrivateHost(host) {
		return ErrUnsafeURL
	}
	return nil
}

// SanitizeHeaderValue strips CR/LF from a value destined for an HTTP
// header, preventing header-injection attacks.
func SanitizeHeaderValue(v string) string {
	if v == "" {
		return v
	}
	repl := strings.NewReplacer("\r", "", "\n", "")
	return repl.Replace(v)
}

// RequestIDFromHeaders extracts a request ID following the correlation
// convention ("X-Request-Id"), or generates one when absent.
func RequestIDFromHeaders(h http.Header) string {
	if v := h.Get("X-Request-Id"); v != "" {
		return SanitizeHeaderValue(v)
	}
	return NewRequestID()
}

// ResponseWriterNoCache is a small wrapper that sets no-store headers.
// It is used by middleware to prevent credential-bearing responses from
// being cached by proxies or browsers.
func ResponseWriterNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

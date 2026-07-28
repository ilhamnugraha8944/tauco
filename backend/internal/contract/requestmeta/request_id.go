// Package requestmeta owns the transport-neutral request ID contract shared by
// middleware and generated API delivery.
package requestmeta

import "strings"

const (
	// Header is the canonical request/response trace header.
	Header = "X-Request-ID"
	// MaxLength is the public contract bound for caller-provided request IDs.
	MaxLength = 128
)

// Valid reports whether a request ID is safe for headers, logs, and public
// response metadata.
func Valid(requestID string) bool {
	if requestID == "" ||
		len(requestID) > MaxLength ||
		requestID != strings.TrimSpace(requestID) {
		return false
	}

	for _, character := range requestID {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_', character == '.', character == ':':
		default:
			return false
		}
	}
	return true
}

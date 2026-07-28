package requestmeta

import (
	"net/http"
	"strings"
)

const (
	// MaxProblemInstanceLength bounds the escaped path copied to public
	// RFC 7807 responses.
	MaxProblemInstanceLength = 2048
)

// ProblemInstancePath returns a bounded absolute, escaped request path suitable
// for the RFC 7807 instance member. Unsafe or unavailable paths fall back to
// the root reference; requestId remains the occurrence correlation key.
func ProblemInstancePath(request *http.Request) string {
	if request == nil || request.URL == nil {
		return "/"
	}

	instance := request.URL.EscapedPath()
	if instance == "" ||
		len(instance) > MaxProblemInstanceLength ||
		!strings.HasPrefix(instance, "/") ||
		strings.HasPrefix(instance, "//") {
		return "/"
	}

	return instance
}

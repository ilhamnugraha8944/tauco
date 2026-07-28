package requestmeta_test

import (
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

func TestValidRequestID(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"01J00000000000000000000001",
		"upstream.request-123",
		"service_name:trace",
	} {
		if !requestmeta.Valid(value) {
			t.Errorf("Valid(%q) = false, want true", value)
		}
	}

	for _, value := range []string{
		"",
		" unsafe",
		"unsafe request id",
		"visitor@example.com",
		"line\nbreak",
		strings.Repeat("a", requestmeta.MaxLength+1),
	} {
		if requestmeta.Valid(value) {
			t.Errorf("Valid(%q) = true, want false", value)
		}
	}
}

package logging

import (
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestStandardLoggerRedactsPayload(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.DebugLevel)
	logger := newLogger(core, "tauco-api", config.EnvironmentTest)

	logger.StandardLogger("http").Print(
		"connection failed for person@example.com token=secret-value",
	)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	serialized := entries[0].Message + " " + entries[0].ContextMap()["error"].(string)
	for _, forbidden := range []string{"person@example.com", "secret-value"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("standard logger leaked %q: %s", forbidden, serialized)
		}
	}
}

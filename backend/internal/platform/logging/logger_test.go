package logging

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerAddsBaseFieldsAndAllowedContext(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.DebugLevel)
	logger := newLogger(core, "tauco-api", config.EnvironmentTest)

	logger.Info(
		"request completed",
		RequestID("req-123"),
		Route("/api/v1/products/:slug"),
		Method("GET"),
		Status(200),
		Latency(1250*time.Microsecond),
	)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	context := entries[0].ContextMap()
	assertContextValue(t, context, "service", "tauco-api")
	assertContextValue(t, context, "environment", "test")
	assertContextValue(t, context, "request_id", "req-123")
	assertContextValue(t, context, "route", "/api/v1/products/:slug")
	assertContextValue(t, context, "method", "GET")
	assertContextValue(t, context, "status", int64(200))
	assertContextValue(t, context, "latency_ms", int64(1))
}

func TestLoggerRedactsMessagesAndErrors(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.DebugLevel)
	logger := newLogger(core, "tauco-api", config.EnvironmentTest)

	logger.Error(
		"request from admin@example.com with Authorization: Bearer visible-token",
		Cause(errors.New("database postgres://owner:hunter2@db.example.test/tauco token=abc123 phone=+62 812 3456 7890 ip=192.168.1.10")),
	)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	serialized := entries[0].Message + " " + entries[0].ContextMap()["error"].(string)
	for _, forbidden := range []string{
		"admin@example.com",
		"visible-token",
		"owner:hunter2",
		"abc123",
		"+62 812 3456 7890",
		"192.168.1.10",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Errorf("log output contains forbidden value %q: %s", forbidden, serialized)
		}
	}
}

func TestCauseNilIsSkipped(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.DebugLevel)
	logger := newLogger(core, "tauco-api", config.EnvironmentTest)
	logger.Info("operation completed", Cause(nil), Field{})

	context := observed.All()[0].ContextMap()
	if _, found := context["error"]; found {
		t.Fatalf("nil Cause() emitted an error field: %#v", context)
	}
}

func TestNamedAndWithPreserveContext(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.DebugLevel)
	logger := newLogger(core, "tauco-api", config.EnvironmentTest).
		Named("http middleware").
		With(Component("delivery"))

	logger.Warn("rate limit fallback", CacheOutcome("fallback"))

	entry := observed.All()[0]
	if entry.LoggerName != "httpmiddleware" {
		t.Errorf("LoggerName = %q, want sanitized name", entry.LoggerName)
	}
	context := entry.ContextMap()
	assertContextValue(t, context, "component", "delivery")
	assertContextValue(t, context, "cache_outcome", "fallback")
}

func TestRedactStringCoversSensitiveRepresentations(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`{"password":"hunter2","email":"person@example.com"}`,
		"Authorization: Basic dXNlcjpwYXNz",
		"access_token=secret-token",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.signature",
		"redis://default:secret@redis.example.test:6379",
		"caller 10.0.0.1 [2001:db8::1] +62 (812) 3456-7890",
	}

	for _, input := range inputs {
		output := RedactString(input)
		for _, forbidden := range []string{
			"hunter2",
			"person@example.com",
			"dXNlcjpwYXNz",
			"secret-token",
			"eyJhbGciOiJIUzI1NiJ9",
			"default:secret",
			"10.0.0.1",
			"2001:db8::1",
			"812) 3456",
		} {
			if strings.Contains(output, forbidden) {
				t.Errorf("RedactString(%q) leaked %q: %q", input, forbidden, output)
			}
		}
	}
}

func TestRedactStringEscapesLineBreaksAndBoundsLength(t *testing.T) {
	t.Parallel()

	output := RedactString(strings.Repeat("x", maxLogTextLen+50) + "\nforged")
	if strings.ContainsAny(output, "\r\n") {
		t.Fatalf("output contains a physical line break")
	}
	if !strings.HasSuffix(output, "...[TRUNCATED]") {
		t.Fatalf("output was not marked as truncated")
	}
}

func TestNewRejectsUnsupportedConfigWithoutEchoingValue(t *testing.T) {
	t.Parallel()

	_, err := New(config.LogConfig{
		Level:   "trace-secret-value",
		Format:  "json",
		Service: "tauco-api",
	}, config.EnvironmentLocal)
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}
	if strings.Contains(err.Error(), "trace-secret-value") {
		t.Fatalf("error leaked rejected value: %v", err)
	}

	_, err = New(config.LogConfig{
		Level:   "info",
		Format:  "xml-secret-value",
		Service: "tauco-api",
	}, config.EnvironmentLocal)
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}
	if strings.Contains(err.Error(), "xml-secret-value") {
		t.Fatalf("error leaked rejected value: %v", err)
	}
}

func assertContextValue(t *testing.T, context map[string]any, key string, want any) {
	t.Helper()

	got, found := context[key]
	if !found {
		t.Errorf("context does not contain %q: %#v", key, context)
		return
	}
	if got != want {
		t.Errorf("context[%q] = %#v, want %#v", key, got, want)
	}
}

package composition

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
)

func TestNewComposesLivenessRoute(t *testing.T) {
	app, err := New(testConfig())
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	app.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want ok", body["status"])
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	cfg := testConfig()
	cfg.HTTP.Port = 0

	if _, err := New(cfg); err == nil {
		t.Fatal("New() error = nil, want validation error")
	}
}

func testConfig() config.Config {
	return config.Config{
		Environment: config.EnvironmentTest,
		HTTP: config.HTTPConfig{
			Host:                "127.0.0.1",
			Port:                8080,
			ReadHeaderTimeout:   time.Second,
			ReadTimeout:         2 * time.Second,
			WriteTimeout:        2 * time.Second,
			IdleTimeout:         5 * time.Second,
			ShutdownGracePeriod: time.Second,
		},
		Log: config.LogConfig{
			Level:   "error",
			Format:  "json",
			Service: "tauco-api",
		},
	}
}
